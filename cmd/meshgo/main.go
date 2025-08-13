package main

import (
	"context"
	"errors"
	"log/slog"
	"net"
	"os"
	"path/filepath"
	"strconv"

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
	ctx := context.Background()

	cfgDir, err := os.UserConfigDir()
	if err != nil {
		slog.Error("config dir", "err", err)
		os.Exit(1)
	}
	cfgDir = filepath.Join(cfgDir, "meshgo")
	if err := os.MkdirAll(cfgDir, 0o755); err != nil {
		slog.Error("create config dir", "err", err)
		os.Exit(1)
	}
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
		}
	}

	logDir := filepath.Join(cfgDir, "logs")
	if err := os.MkdirAll(logDir, 0o755); err != nil {
		slog.Error("create log dir", "err", err)
		os.Exit(1)
	}
	logPath := filepath.Join(logDir, "meshgo.log")
	l, closer, err := logger.New(logPath, settings.Logging.Enabled)
	if err != nil {
		slog.Error("init logger", "err", err)
		os.Exit(1)
	}
	defer closer.Close()
	slog.SetDefault(l)

	var t transport.Transport
	switch settings.Connection.Type {
	case "serial":
		t = transport.NewSerial(settings.Connection.Serial.Port)
	default:
		host := settings.Connection.IP.Host
		if host == "" {
			host = "localhost"
		}
		port := settings.Connection.IP.Port
		if port == 0 {
			port = 4403
		}
		t = transport.NewTCP(net.JoinHostPort(host, strconv.Itoa(port)))
	}

	dbPath := filepath.Join(cfgDir, "meshgo.db")
	ms, err := storage.OpenMessageStore(dbPath)
	if err != nil {
		slog.Error("open message store", "err", err)
		os.Exit(1)
	}
	defer ms.Close()
	if err := ms.Init(ctx); err != nil {
		slog.Error("init message store", "err", err)
		os.Exit(1)
	}

	ns, err := storage.OpenNodeStore(dbPath)
	if err != nil {
		slog.Error("open node store", "err", err)
		os.Exit(1)
	}
	defer ns.Close()
	if err := ns.Init(ctx); err != nil {
		slog.Error("init node store", "err", err)
		os.Exit(1)
	}

	cs, err := storage.OpenChatStore(dbPath)
	if err != nil {
		slog.Error("open chat store", "err", err)
		os.Exit(1)
	}
	defer cs.Close()
	if err := cs.Init(ctx); err != nil {
		slog.Error("init chat store", "err", err)
		os.Exit(1)
	}

	rc := radio.New(radio.ReconnectConfig{
		InitialMillis: settings.Reconnect.InitialMillis,
		MaxMillis:     settings.Reconnect.MaxMillis,
		Multiplier:    settings.Reconnect.Multiplier,
		Jitter:        settings.Reconnect.Jitter,
	})

	notifier := notify.NewBeeep(settings.Notifications.Enabled)

	tr := &tray.Noop{}
	go tr.Run()
	a := app.New(rc, ms, ns, cs, notifier, tr)
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()
	if err := a.Run(ctx, t); err != nil && !errors.Is(err, context.Canceled) {
		slog.Error("run app", "err", err)
		os.Exit(1)
	}
}
