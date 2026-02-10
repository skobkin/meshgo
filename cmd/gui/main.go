package main

import (
	"context"
	"log/slog"
	"os"
	"os/signal"
	"sync"
	"syscall"

	"github.com/skobkin/meshgo/internal/app"
	"github.com/skobkin/meshgo/internal/ui"
)

func main() {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	rt, err := app.Initialize(ctx)
	if err != nil {
		slog.Error("initialize app runtime", "error", err)
		os.Exit(1)
	}

	var closeOnce sync.Once
	closeRuntime := func() {
		closeOnce.Do(func() {
			_ = rt.Close()
		})
	}
	defer closeRuntime()

	err = ui.Run(ui.Dependencies{
		Config:      rt.Config,
		ChatStore:   rt.ChatStore,
		NodeStore:   rt.NodeStore,
		Bus:         rt.Bus,
		Sender:      rt.Radio,
		IPTransport: rt.IPTransport,
		OnSave:      rt.SaveAndApplyConfig,
		OnClearDB:   rt.ClearDatabase,
		OnQuit: func() {
			stop()
			closeRuntime()
		},
	})
	if err != nil {
		slog.Error("run ui", "error", err)
		os.Exit(1)
	}
}
