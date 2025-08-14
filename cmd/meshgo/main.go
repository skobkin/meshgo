package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"log/slog"
	"net"
	"os"
	"path/filepath"
	"strconv"
	"time"

	fyneapp "fyne.io/fyne/v2/app"
	"fyne.io/fyne/v2/widget"

	"meshgo/app"
	"meshgo/logger"
	"meshgo/notify"
	"meshgo/radio"
	"meshgo/storage"
	"meshgo/transport"
	"meshgo/tray"
)

// main starts the meshgo application. This is a minimal placeholder that
// demonstrates wiring the core app with a transport.
func main() {
	listChats := flag.Bool("list-chats", false, "list stored chats and exit")
	listNodes := flag.Bool("list-nodes", false, "list stored nodes and exit")
	listChannels := flag.Bool("list-channels", false, "list stored channels and exit")
	listMessages := flag.String("list-messages", "", "list messages for the given chat and exit")
	flag.Parse()

	ctx := context.Background()

	cfgDir, err := os.UserConfigDir()
	if err != nil {
		slog.Error("config dir", "err", err)
		os.Exit(1)
	}
	cfgDir = filepath.Join(cfgDir, "meshgo")
	if err := os.MkdirAll(cfgDir, 0755); err != nil {
		slog.Error("create config dir", "err", err)
		os.Exit(1)
	}
	slog.Info("using config dir", "path", cfgDir)
	settingsPath := filepath.Join(cfgDir, "config.json")

	settings, err := storage.LoadSettings(settingsPath)
	if err != nil && !errors.Is(err, os.ErrNotExist) {
		slog.Error("load settings", "err", err)
		os.Exit(1)
	}
	if settings == nil {
		settings = &storage.Settings{}
		if err := storage.SaveSettings(settingsPath, settings); err != nil {
			slog.Error("save default settings", "err", err)
		} else {
			slog.Info("wrote default settings", "path", settingsPath)
		}
	} else {
		slog.Info("loaded settings", "path", settingsPath)
	}

	logDir := filepath.Join(cfgDir, "logs")
	if err := os.MkdirAll(logDir, 0755); err != nil {
		slog.Error("create log dir", "err", err)
		os.Exit(1)
	}
	slog.Info("using log dir", "path", logDir)
	logPath := filepath.Join(logDir, "meshgo.log")
	l, closer, err := logger.New(logPath, settings.Logging.Enabled)
	if err != nil {
		slog.Error("init logger", "err", err)
		os.Exit(1)
	}
	defer closer.Close()
	slog.SetDefault(l)
	slog.Info("starting meshgo", "log", logPath)

	var (
		t       transport.Transport
		hasConn bool
	)
	switch settings.Connection.Type {
	case "serial":
		if settings.Connection.Serial.Port != "" {
			t = transport.NewSerial(settings.Connection.Serial.Port)
			slog.Info("configured transport", "type", "serial", "endpoint", t.Endpoint())
			hasConn = true
		}
	case "tcp":
		fallthrough
	default:
		host := settings.Connection.IP.Host
		port := settings.Connection.IP.Port
		if host != "" || port != 0 || settings.Connection.Type == "tcp" {
			if host == "" {
				host = "localhost"
			}
			if port == 0 {
				port = 4403
			}
			t = transport.NewTCP(net.JoinHostPort(host, strconv.Itoa(port)))
			slog.Info("configured transport", "type", "tcp", "endpoint", t.Endpoint())
			hasConn = true
		}
	}
	if !hasConn {
		slog.Info("no connection configured; radio disabled")
	}

	dbPath := filepath.Join(cfgDir, "meshgo.db")
	ms, err := storage.OpenMessageStore(dbPath)
	if err != nil {
		slog.Error("open message store", "err", err)
		os.Exit(1)
	}
	slog.Info("opened message store", "db", dbPath)
	defer ms.Close()
	if err := ms.Init(ctx); err != nil {
		slog.Error("init message store", "err", err)
		os.Exit(1)
	}
	slog.Info("initialized message store")

	ns, err := storage.OpenNodeStore(dbPath)
	if err != nil {
		slog.Error("open node store", "err", err)
		os.Exit(1)
	}
	slog.Info("opened node store")
	defer ns.Close()
	if err := ns.Init(ctx); err != nil {
		slog.Error("init node store", "err", err)
		os.Exit(1)
	}
	slog.Info("initialized node store")

	cs, err := storage.OpenChatStore(dbPath)
	if err != nil {
		slog.Error("open chat store", "err", err)
		os.Exit(1)
	}
	slog.Info("opened chat store")
	defer cs.Close()
	if err := cs.Init(ctx); err != nil {
		slog.Error("init chat store", "err", err)
		os.Exit(1)
	}
	slog.Info("initialized chat store")

	chs, err := storage.OpenChannelStore(dbPath)
	if err != nil {
		slog.Error("open channel store", "err", err)
		os.Exit(1)
	}
	slog.Info("opened channel store")
	defer chs.Close()
	if err := chs.Init(ctx); err != nil {
		slog.Error("init channel store", "err", err)
		os.Exit(1)
	}
	slog.Info("initialized channel store")

	a := app.New(nil, ms, ns, cs, chs, nil, nil)
	if *listChats {
		chats, err := a.ListChats(ctx)
		if err != nil {
			slog.Error("list chats", "err", err)
			os.Exit(1)
		}
		for _, c := range chats {
			fmt.Printf("%s (unread: %d)\n", c.Title, c.UnreadCount)
		}
		return
	}
	if *listNodes {
		nodes, err := a.ListNodes(ctx)
		if err != nil {
			slog.Error("list nodes", "err", err)
			os.Exit(1)
		}
		for _, n := range nodes {
			fmt.Printf("%s (%s)\n", n.ID, n.ShortName)
		}
		return
	}
	if *listChannels {
		chans, err := a.ListChannels(ctx)
		if err != nil {
			slog.Error("list channels", "err", err)
			os.Exit(1)
		}
		for _, c := range chans {
			fmt.Printf("%s (psk=%d)\n", c.Name, c.PSKClass)
		}
		return
	}
	if *listMessages != "" {
		msgs, err := a.ListMessages(ctx, *listMessages, 100)
		if err != nil {
			slog.Error("list messages", "err", err)
			os.Exit(1)
		}
		for _, m := range msgs {
			fmt.Printf("%s: %s\n", m.Timestamp.Format(time.RFC3339), m.Text)
		}
		return
	}

	rc := radio.New(radio.ReconnectConfig{
		InitialMillis: settings.Reconnect.InitialMillis,
		MaxMillis:     settings.Reconnect.MaxMillis,
		Multiplier:    settings.Reconnect.Multiplier,
		Jitter:        settings.Reconnect.Jitter,
	})

	notifier := notify.NewBeeep(settings.Notifications.Enabled)

	tr := tray.NewSystray(settings.Notifications.Enabled)

	uiApp := fyneapp.New()
	win := uiApp.NewWindow("meshgo")
	win.SetContent(widget.NewLabel("meshgo running"))
	win.SetCloseIntercept(func() { win.Hide() })

	visible := true
	tr.OnShowHide(func() {
		if visible {
			win.Hide()
		} else {
			win.Show()
		}
		visible = !visible
	})

	tr.OnToggleNotifications(func(e bool) {
		notifier.SetEnabled(e)
		settings.Notifications.Enabled = e
		if err := storage.SaveSettings(settingsPath, settings); err != nil {
			slog.Error("save settings", "err", err)
		}
	})

	ctx, cancel := context.WithCancel(ctx)
	tr.OnExit(func() {
		cancel()
		uiApp.Quit()
	})
	defer cancel()

	a = app.New(rc, ms, ns, cs, chs, notifier, tr)

	go func() {
		for {
			select {
			case ev, ok := <-a.Events():
				if !ok {
					return
				}
				switch ev.Type {
				case app.EventConnecting:
					slog.Info("connecting")
				case app.EventConnected:
					slog.Info("connected")
				case app.EventDisconnected:
					slog.Info("disconnected", "err", ev.Err)
				case app.EventRetrying:
					slog.Info("retrying", "delay", ev.Delay)
				case app.EventMessage:
					if ev.Message != nil {
						slog.Info("message", "chat", ev.Message.ChatID, "text", ev.Message.Text)
					}
				case app.EventNode:
					if ev.Node != nil {
						slog.Info("node", "id", ev.Node.ID, "short", ev.Node.ShortName, "signal", ev.Node.Signal)
					}
				}
			case <-ctx.Done():
				return
			}
		}
	}()

	tr.OnReady(func() {
		slog.Info("tray ready")
		if !hasConn {
			slog.Info("no connection configured; radio not started")
			return
		}
		slog.Info("starting radio", "endpoint", t.Endpoint())
		go func() {
			if err := a.Run(ctx, t); err != nil && !errors.Is(err, context.Canceled) {
				slog.Error("run app", "err", err)
			}
			cancel()
			tr.Quit()
		}()
	})

	go tr.Run()
	win.Show()
	uiApp.Run()
}
