package main

import (
	"context"
	"flag"
	"fmt"
	"log/slog"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/skobkin/meshgo/internal/app"
	"github.com/skobkin/meshgo/internal/bus"
	"github.com/skobkin/meshgo/internal/config"
	"github.com/skobkin/meshgo/internal/connectors"
	"github.com/skobkin/meshgo/internal/domain"
	"github.com/skobkin/meshgo/internal/logging"
	"github.com/skobkin/meshgo/internal/persistence"
	"github.com/skobkin/meshgo/internal/radio"
	"github.com/skobkin/meshgo/internal/transport"
)

const (
	initialConfigWaitTimeout = 45 * time.Second
	maxHexPreviewLen         = 64
)

func main() {
	if err := run(); err != nil {
		slog.Error("run debug tool", "error", err)
		os.Exit(1)
	}
}

func run() error {
	connector := flag.String("connector", "ip", "connector type (draft: ip)")
	host := flag.String("host", "", "ip/hostname")
	noSubscribe := flag.Bool("no-subscribe", false, "exit after initial config download completes")
	listenFor := flag.Duration("listen-for", 0, "listen duration, e.g. 30s")
	flag.Parse()

	if *connector != "ip" {
		return fmt.Errorf("only ip connector is supported in draft: %s", *connector)
	}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	paths, err := app.ResolvePaths()
	if err != nil {
		return fmt.Errorf("resolve paths: %w", err)
	}
	cfg, err := config.Load(paths.ConfigFile)
	if err != nil {
		return fmt.Errorf("load config: %w", err)
	}
	if strings.TrimSpace(*host) != "" {
		cfg.Connection.Host = strings.TrimSpace(*host)
	}
	if strings.TrimSpace(cfg.Connection.Host) == "" {
		return fmt.Errorf("missing ip host: set --host or save connection host in config")
	}

	logMgr := logging.NewManager()
	cfg.Logging.LogToFile = false
	if err := logMgr.Configure(cfg.Logging, paths.LogFile); err != nil {
		return fmt.Errorf("configure logging: %w", err)
	}
	defer func() {
		if closeErr := logMgr.Close(); closeErr != nil {
			slog.Warn("close log manager", "error", closeErr)
		}
	}()
	logger := logMgr.Logger("cli")
	logger.Info("starting meshgo debug", "version", app.BuildVersion(), "build_date", app.BuildDateYMD())

	db, err := persistence.Open(ctx, paths.DBFile)
	if err != nil {
		return fmt.Errorf("open sqlite: %w", err)
	}
	defer func() {
		if closeErr := db.Close(); closeErr != nil {
			logger.Warn("close sqlite", "error", closeErr)
		}
	}()

	nodeRepo := persistence.NewNodeRepo(db)
	chatRepo := persistence.NewChatRepo(db)
	msgRepo := persistence.NewMessageRepo(db)

	nodes, err := nodeRepo.ListSortedByLastHeard(ctx)
	if err != nil {
		return fmt.Errorf("load cached nodes: %w", err)
	}
	chats, err := chatRepo.ListSortedByLastSentByMe(ctx)
	if err != nil {
		return fmt.Errorf("load cached chats: %w", err)
	}
	logger.Info("cached state", "nodes", len(nodes), "chats", len(chats))

	b := bus.New(logMgr.Logger("bus"))
	defer b.Close()

	nodeStore := domain.NewNodeStore()
	chatStore := domain.NewChatStore()
	if err := domain.LoadStoresFromPersistence(ctx, nodeStore, chatStore, nodeRepo, chatRepo, msgRepo); err != nil {
		return fmt.Errorf("bootstrap stores: %w", err)
	}
	nodeStore.Start(ctx, b)
	chatStore.Start(ctx, b)

	writer := persistence.NewWriterQueue(logMgr.Logger("persistence"), 256)
	writer.Start(ctx)
	domain.StartPersistenceSync(ctx, b, writer, nodeRepo, chatRepo, msgRepo)

	codec, err := radio.NewMeshtasticCodec()
	if err != nil {
		return fmt.Errorf("initialize meshtastic codec: %w", err)
	}

	tr := transport.NewIPTransport(cfg.Connection.Host, app.DefaultIPPort)
	radioSvc := radio.NewService(logMgr.Logger("radio"), b, tr, codec)
	initialDecodedSub := b.Subscribe(connectors.TopicRadioFrom)
	initialConnSub := b.Subscribe(connectors.TopicConnStatus)
	initialRawInSub := b.Subscribe(connectors.TopicRawFrameIn)
	initialRawOutSub := b.Subscribe(connectors.TopicRawFrameOut)
	defer b.Unsubscribe(initialDecodedSub, connectors.TopicRadioFrom)
	defer b.Unsubscribe(initialConnSub, connectors.TopicConnStatus)
	defer b.Unsubscribe(initialRawInSub, connectors.TopicRawFrameIn)
	defer b.Unsubscribe(initialRawOutSub, connectors.TopicRawFrameOut)
	radioSvc.Start(ctx)

	logger.Info("waiting for initial config completion", "host", cfg.Connection.Host, "timeout", initialConfigWaitTimeout)
	if err := waitForInitialConfig(ctx, logger, initialDecodedSub, initialConnSub, initialRawInSub, initialRawOutSub, initialConfigWaitTimeout); err != nil {
		return fmt.Errorf("initial config did not complete: %w", err)
	}
	logger.Info("initial config completed")
	logInitialSnapshot(logger, nodeStore, chatStore)

	if *noSubscribe {
		logger.Info("no-subscribe mode completed, exiting")
		return nil
	}

	watch(ctx, b, logger)

	if *listenFor > 0 {
		logger.Info("listen mode", "duration", *listenFor)
		select {
		case <-ctx.Done():
		case <-time.After(*listenFor):
		}
		return nil
	}

	logger.Info("listening until interrupt")
	<-ctx.Done()

	return nil
}

func waitForInitialConfig(ctx context.Context, logger *slog.Logger, decodedSub, connSub, rawInSub, rawOutSub bus.Subscription, timeout time.Duration) error {
	var inFrames, outFrames, decodedFrames int
	timeoutCh := time.After(timeout)
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-timeoutCh:
			logger.Info("initial phase summary", "in_frames", inFrames, "out_frames", outFrames, "decoded_frames", decodedFrames)
			return fmt.Errorf("timeout waiting for config_complete_id response after %s", timeout)
		case raw, ok := <-connSub:
			if !ok {
				continue
			}
			status, ok := raw.(connectors.ConnStatus)
			if !ok {
				continue
			}
			logger.Info("initial conn", "state", status.State, "transport", status.TransportName, "error", status.Err)
		case raw, ok := <-rawOutSub:
			if !ok {
				continue
			}
			frame, ok := raw.(connectors.RawFrame)
			if !ok {
				continue
			}
			outFrames++
			logger.Info("initial raw out", "len", frame.Len, "hex", previewHex(frame.Hex))
		case raw, ok := <-rawInSub:
			if !ok {
				continue
			}
			frame, ok := raw.(connectors.RawFrame)
			if !ok {
				continue
			}
			inFrames++
			logger.Info("initial raw in", "len", frame.Len, "hex", previewHex(frame.Hex))
		case raw, ok := <-decodedSub:
			if !ok {
				return fmt.Errorf("radio stream closed while waiting for initial config")
			}
			frame, ok := raw.(radio.DecodedFrame)
			if !ok {
				continue
			}
			decodedFrames++
			logger.Info(
				"initial decoded",
				"raw_len", len(frame.Raw),
				"config_complete_id", frame.ConfigCompleteID,
				"want_config_ready", frame.WantConfigReady,
				"has_node", frame.NodeUpdate != nil,
				"has_channels", frame.Channels != nil,
				"has_text", frame.TextMessage != nil,
				"has_config", frame.ConfigSnapshot != nil,
			)
			if frame.WantConfigReady {
				logger.Info("initial phase summary", "in_frames", inFrames, "out_frames", outFrames, "decoded_frames", decodedFrames)
				return nil
			}
		}
	}
}

func watch(ctx context.Context, b bus.MessageBus, logger *slog.Logger) {
	connSub := b.Subscribe(connectors.TopicConnStatus)
	channelSub := b.Subscribe(connectors.TopicChannels)
	nodeSub := b.Subscribe(connectors.TopicNodeInfo)
	textSub := b.Subscribe(connectors.TopicTextMessage)
	statusSub := b.Subscribe(connectors.TopicMessageStatus)
	configSub := b.Subscribe(connectors.TopicConfigSnapshot)
	rawInSub := b.Subscribe(connectors.TopicRawFrameIn)
	rawOutSub := b.Subscribe(connectors.TopicRawFrameOut)

	go func() {
		for {
			select {
			case <-ctx.Done():
				b.Unsubscribe(connSub, connectors.TopicConnStatus)
				b.Unsubscribe(channelSub, connectors.TopicChannels)
				b.Unsubscribe(nodeSub, connectors.TopicNodeInfo)
				b.Unsubscribe(textSub, connectors.TopicTextMessage)
				b.Unsubscribe(statusSub, connectors.TopicMessageStatus)
				b.Unsubscribe(configSub, connectors.TopicConfigSnapshot)
				b.Unsubscribe(rawInSub, connectors.TopicRawFrameIn)
				b.Unsubscribe(rawOutSub, connectors.TopicRawFrameOut)
				return
			case raw := <-connSub:
				if status, ok := raw.(connectors.ConnStatus); ok {
					logger.Info("conn", "state", status.State, "transport", status.TransportName, "error", status.Err)
				}
			case raw := <-channelSub:
				if channels, ok := raw.(domain.ChannelList); ok {
					logger.Info("channels", "count", len(channels.Items))
				}
			case raw := <-nodeSub:
				if node, ok := raw.(domain.NodeUpdate); ok {
					logger.Info("node", "id", node.Node.NodeID, "name", node.Node.LongName)
				}
			case raw := <-textSub:
				if msg, ok := raw.(domain.ChatMessage); ok {
					logger.Info("text", "chat", msg.ChatKey, "direction", msg.Direction, "body", msg.Body)
				}
			case raw := <-statusSub:
				if update, ok := raw.(domain.MessageStatusUpdate); ok {
					logger.Info("message-status", "device_message_id", update.DeviceMessageID, "status", update.Status, "reason", update.Reason)
				}
			case raw := <-configSub:
				if cfg, ok := raw.(connectors.ConfigSnapshot); ok {
					logger.Info("config-snapshot", "channels", fmt.Sprintf("%v", cfg.ChannelTitles))
				}
			case raw := <-rawOutSub:
				if frame, ok := raw.(connectors.RawFrame); ok {
					logger.Info("raw-out", "len", frame.Len, "hex", previewHex(frame.Hex))
				}
			case raw := <-rawInSub:
				if frame, ok := raw.(connectors.RawFrame); ok {
					logger.Info("raw-in", "len", frame.Len, "hex", previewHex(frame.Hex))
				}
			}
		}
	}()
}

func logInitialSnapshot(logger *slog.Logger, nodeStore *domain.NodeStore, chatStore *domain.ChatStore) {
	nodes := nodeStore.SnapshotSorted()
	logger.Info("node summary", "count", len(nodes))
	for i, node := range nodes {
		if i >= 10 {
			logger.Info("node summary truncated", "remaining", len(nodes)-i)
			break
		}
		name := strings.TrimSpace(node.LongName)
		if name == "" {
			name = node.NodeID
		}
		logger.Info("node item", "id", node.NodeID, "name", name, "heard", node.LastHeardAt.Format(time.RFC3339))
	}

	chats := chatStore.ChatListSorted()
	channels := make([]string, 0, len(chats))
	for _, chat := range chats {
		if chat.Type != domain.ChatTypeChannel {
			continue
		}
		channels = append(channels, chat.Title)
	}
	logger.Info("channels summary", "count", len(channels), "titles", fmt.Sprintf("%v", channels))
}

func previewHex(hex string) string {
	hex = strings.TrimSpace(hex)
	if len(hex) <= maxHexPreviewLen {
		return hex
	}
	return hex[:maxHexPreviewLen] + "..."
}
