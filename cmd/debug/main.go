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
	"github.com/skobkin/meshgo/internal/domain"
	"github.com/skobkin/meshgo/internal/logging"
	"github.com/skobkin/meshgo/internal/persistence"
	"github.com/skobkin/meshgo/internal/radio"
	"github.com/skobkin/meshgo/internal/radio/busmsg"
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
	transportFlag := flag.String("transport", "", "transport type (ip|serial|bluetooth); defaults to config value")
	host := flag.String("host", "", "ip/hostname")
	serialPort := flag.String("serial-port", "", "serial port path/name (example: /dev/ttyACM0, COM3)")
	serialBaud := flag.Int("serial-baud", 0, "serial baud rate (example: 115200)")
	bluetoothAddress := flag.String("bluetooth-address", "", "bluetooth device address (example: AA:BB:CC:DD:EE:FF)")
	bluetoothAdapter := flag.String("bluetooth-adapter", "", "bluetooth adapter id (example: hci0)")
	noSubscribe := flag.Bool("no-subscribe", false, "exit after initial config download completes")
	listenFor := flag.Duration("listen-for", 0, "listen duration, e.g. 30s")
	flag.Parse()

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
	cfg.FillMissingDefaults()

	selectedTransport := strings.ToLower(strings.TrimSpace(*transportFlag))
	if selectedTransport != "" {
		cfg.Connection.Transport = config.TransportType(selectedTransport)
	}
	if strings.TrimSpace(*host) != "" {
		cfg.Connection.Host = strings.TrimSpace(*host)
	}
	if strings.TrimSpace(*serialPort) != "" {
		cfg.Connection.SerialPort = strings.TrimSpace(*serialPort)
	}
	if *serialBaud > 0 {
		cfg.Connection.SerialBaud = *serialBaud
	}
	if strings.TrimSpace(*bluetoothAddress) != "" {
		cfg.Connection.BluetoothAddress = strings.TrimSpace(*bluetoothAddress)
	}
	if strings.TrimSpace(*bluetoothAdapter) != "" {
		cfg.Connection.BluetoothAdapter = strings.TrimSpace(*bluetoothAdapter)
	}

	if err := cfg.Validate(); err != nil {
		return fmt.Errorf("invalid connection config: %w", err)
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

	nodeCoreRepo := persistence.NewNodeCoreRepo(db)
	nodePositionRepo := persistence.NewNodePositionRepo(db)
	nodeTelemetryRepo := persistence.NewNodeTelemetryRepo(db)
	chatRepo := persistence.NewChatRepo(db)
	msgRepo := persistence.NewMessageRepo(db)
	tracerouteRepo := persistence.NewTracerouteRepo(db)

	nodesCore, err := nodeCoreRepo.ListSortedByLastHeard(ctx)
	if err != nil {
		return fmt.Errorf("load cached nodes: %w", err)
	}
	nodesPosition, err := nodePositionRepo.ListLatest(ctx)
	if err != nil {
		return fmt.Errorf("load cached node positions: %w", err)
	}
	nodesTelemetry, err := nodeTelemetryRepo.ListLatest(ctx)
	if err != nil {
		return fmt.Errorf("load cached node telemetry: %w", err)
	}
	nodes := domainMergeNodesForDebug(nodesCore, nodesPosition, nodesTelemetry)
	chats, err := chatRepo.ListSortedByLastSentByMe(ctx)
	if err != nil {
		return fmt.Errorf("load cached chats: %w", err)
	}
	logger.Info("cached state", "nodes", len(nodes), "chats", len(chats))

	b := bus.New(logMgr.Logger("bus"))
	defer b.Close()

	nodeStore := domain.NewNodeStore()
	chatStore := domain.NewChatStore()
	if err := domain.LoadStoresFromRepositories(ctx, nodeStore, chatStore, nodeCoreRepo, nodePositionRepo, nodeTelemetryRepo, chatRepo, msgRepo); err != nil {
		return fmt.Errorf("bootstrap stores: %w", err)
	}
	nodeStore.Start(ctx, b)
	chatStore.Start(ctx, b)

	writer := persistence.NewWriterQueue(logMgr.Logger("persistence"), 256)
	writer.Start(ctx)
	domain.StartPersistenceProjection(
		ctx,
		b,
		writer,
		nodeCoreRepo,
		nodePositionRepo,
		nodeTelemetryRepo,
		debugHistoryLimitsProvider{},
		chatRepo,
		msgRepo,
		tracerouteRepo,
	)

	codec, err := radio.NewMeshtasticCodec()
	if err != nil {
		return fmt.Errorf("initialize meshtastic codec: %w", err)
	}

	connTransport, err := app.NewTransportForConnection(cfg.Connection)
	if err != nil {
		return fmt.Errorf("create transport: %w", err)
	}
	radioService := radio.NewService(logMgr.Logger("radio"), b, connTransport, codec)
	initialDecodedSub := b.Subscribe(bus.TopicRadioFrom)
	initialConnSub := b.Subscribe(bus.TopicConnStatus)
	initialRawInSub := b.Subscribe(bus.TopicRawFrameIn)
	initialRawOutSub := b.Subscribe(bus.TopicRawFrameOut)
	defer b.Unsubscribe(initialDecodedSub, bus.TopicRadioFrom)
	defer b.Unsubscribe(initialConnSub, bus.TopicConnStatus)
	defer b.Unsubscribe(initialRawInSub, bus.TopicRawFrameIn)
	defer b.Unsubscribe(initialRawOutSub, bus.TopicRawFrameOut)
	radioService.Start(ctx)

	logger.Info(
		"waiting for initial config completion",
		"transport",
		cfg.Connection.Transport,
		"target",
		connectionTarget(cfg.Connection),
		"timeout",
		initialConfigWaitTimeout,
	)
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
			status, ok := raw.(busmsg.ConnectionStatus)
			if !ok {
				continue
			}
			logger.Info("initial conn", "state", status.State, "transport", status.TransportName, "error", status.Err)
		case raw, ok := <-rawOutSub:
			if !ok {
				continue
			}
			frame, ok := raw.(busmsg.RawFrame)
			if !ok {
				continue
			}
			outFrames++
			logger.Info("initial raw out", "len", frame.Len, "hex", previewHex(frame.Hex))
		case raw, ok := <-rawInSub:
			if !ok {
				continue
			}
			frame, ok := raw.(busmsg.RawFrame)
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
				"has_node_core", frame.NodeCoreUpdate != nil,
				"has_node_position", frame.NodePositionUpdate != nil,
				"has_node_telemetry", frame.NodeTelemetryUpdate != nil,
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
	connSub := b.Subscribe(bus.TopicConnStatus)
	channelSub := b.Subscribe(bus.TopicChannels)
	nodeCoreSub := b.Subscribe(bus.TopicNodeCore)
	nodePositionSub := b.Subscribe(bus.TopicNodePosition)
	nodeTelemetrySub := b.Subscribe(bus.TopicNodeTelemetry)
	textSub := b.Subscribe(bus.TopicTextMessage)
	statusSub := b.Subscribe(bus.TopicMessageStatus)
	configSub := b.Subscribe(bus.TopicConfigSnapshot)
	rawInSub := b.Subscribe(bus.TopicRawFrameIn)
	rawOutSub := b.Subscribe(bus.TopicRawFrameOut)

	go func() {
		for {
			select {
			case <-ctx.Done():
				b.Unsubscribe(connSub, bus.TopicConnStatus)
				b.Unsubscribe(channelSub, bus.TopicChannels)
				b.Unsubscribe(nodeCoreSub, bus.TopicNodeCore)
				b.Unsubscribe(nodePositionSub, bus.TopicNodePosition)
				b.Unsubscribe(nodeTelemetrySub, bus.TopicNodeTelemetry)
				b.Unsubscribe(textSub, bus.TopicTextMessage)
				b.Unsubscribe(statusSub, bus.TopicMessageStatus)
				b.Unsubscribe(configSub, bus.TopicConfigSnapshot)
				b.Unsubscribe(rawInSub, bus.TopicRawFrameIn)
				b.Unsubscribe(rawOutSub, bus.TopicRawFrameOut)

				return
			case raw := <-connSub:
				if status, ok := raw.(busmsg.ConnectionStatus); ok {
					logger.Info("conn", "state", status.State, "transport", status.TransportName, "error", status.Err)
				}
			case raw := <-channelSub:
				if channels, ok := raw.(domain.ChannelList); ok {
					logger.Info("channels", "count", len(channels.Items))
				}
			case raw := <-nodeCoreSub:
				if node, ok := raw.(domain.NodeCoreUpdate); ok {
					logger.Info("node-core", "id", node.Core.NodeID, "name", domain.NodeDisplayName(domain.Node{
						NodeID:    node.Core.NodeID,
						LongName:  node.Core.LongName,
						ShortName: node.Core.ShortName,
					}))
				}
			case raw := <-nodePositionSub:
				if node, ok := raw.(domain.NodePositionUpdate); ok {
					logger.Info("node-position", "id", node.Position.NodeID, "lat", node.Position.Latitude, "lon", node.Position.Longitude)
				}
			case raw := <-nodeTelemetrySub:
				if node, ok := raw.(domain.NodeTelemetryUpdate); ok {
					logger.Info("node-telemetry", "id", node.Telemetry.NodeID, "battery", node.Telemetry.BatteryLevel)
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
				if cfg, ok := raw.(busmsg.ConfigSnapshot); ok {
					logger.Info("config-snapshot", "channels", fmt.Sprintf("%v", cfg.ChannelTitles))
				}
			case raw := <-rawOutSub:
				if frame, ok := raw.(busmsg.RawFrame); ok {
					logger.Info("raw-out", "len", frame.Len, "hex", previewHex(frame.Hex))
				}
			case raw := <-rawInSub:
				if frame, ok := raw.(busmsg.RawFrame); ok {
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
		name := domain.NodeDisplayName(node)
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

func domainMergeNodesForDebug(coreItems []domain.NodeCore, positionItems []domain.NodePosition, telemetryItems []domain.NodeTelemetry) []domain.Node {
	nodes := make(map[string]domain.Node, len(coreItems))
	for _, core := range coreItems {
		nodes[core.NodeID] = domain.Node{
			NodeID:          core.NodeID,
			LongName:        core.LongName,
			ShortName:       core.ShortName,
			PublicKey:       append([]byte(nil), core.PublicKey...),
			Channel:         core.Channel,
			BoardModel:      core.BoardModel,
			FirmwareVersion: core.FirmwareVersion,
			Role:            core.Role,
			IsUnmessageable: core.IsUnmessageable,
			LastHeardAt:     core.LastHeardAt,
			RSSI:            core.RSSI,
			SNR:             core.SNR,
			UpdatedAt:       core.UpdatedAt,
		}
	}
	for _, pos := range positionItems {
		node := nodes[pos.NodeID]
		node.NodeID = pos.NodeID
		if node.Channel == nil {
			node.Channel = pos.Channel
		}
		node.Latitude = pos.Latitude
		node.Longitude = pos.Longitude
		node.Altitude = pos.Altitude
		node.PositionPrecisionBits = pos.PositionPrecisionBits
		node.PositionUpdatedAt = pos.PositionUpdatedAt
		nodes[pos.NodeID] = node
	}
	for _, tel := range telemetryItems {
		node := nodes[tel.NodeID]
		node.NodeID = tel.NodeID
		if node.Channel == nil {
			node.Channel = tel.Channel
		}
		node.BatteryLevel = tel.BatteryLevel
		node.Voltage = tel.Voltage
		node.UptimeSeconds = tel.UptimeSeconds
		node.ChannelUtilization = tel.ChannelUtilization
		node.AirUtilTx = tel.AirUtilTx
		node.Temperature = tel.Temperature
		node.Humidity = tel.Humidity
		node.Pressure = tel.Pressure
		node.AirQualityIndex = tel.AirQualityIndex
		node.PowerVoltage = tel.PowerVoltage
		node.PowerCurrent = tel.PowerCurrent
		nodes[tel.NodeID] = node
	}

	out := make([]domain.Node, 0, len(nodes))
	for _, node := range nodes {
		out = append(out, node)
	}

	return out
}

type debugHistoryLimitsProvider struct{}

func (debugHistoryLimitsProvider) PositionHistoryLimit() int  { return 100 }
func (debugHistoryLimitsProvider) TelemetryHistoryLimit() int { return 250 }
func (debugHistoryLimitsProvider) IdentityHistoryLimit() int  { return 50 }

func previewHex(hex string) string {
	hex = strings.TrimSpace(hex)
	if len(hex) <= maxHexPreviewLen {
		return hex
	}

	return hex[:maxHexPreviewLen] + "..."
}

func connectionTarget(connection config.ConnectionConfig) string {
	baseTarget := app.ConnectionTarget(connection)

	switch connection.Transport {
	case config.TransportSerial:
		return fmt.Sprintf("%s@%d", baseTarget, connection.SerialBaud)
	case config.TransportBluetooth:
		if baseTarget == "" {
			return ""
		}
		if adapter := strings.TrimSpace(connection.BluetoothAdapter); adapter != "" {
			return fmt.Sprintf("%s (%s)", baseTarget, adapter)
		}

		return baseTarget
	case config.TransportIP:
		return baseTarget
	default:
		return string(connection.Transport)
	}
}
