package main

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"os/signal"
	"sync"
	"syscall"

	"github.com/skobkin/meshgo/internal/app"
	"github.com/skobkin/meshgo/internal/ui"
)

func main() {
	if err := run(); err != nil {
		slog.Error("run gui app", "error", err)
		os.Exit(1)
	}
}

func run() error {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	rt, err := app.Initialize(ctx)
	if err != nil {
		return fmt.Errorf("initialize app runtime: %w", err)
	}

	var closeOnce sync.Once
	closeRuntime := func() {
		closeOnce.Do(func() {
			_ = rt.Close()
		})
	}
	defer closeRuntime()

	err = ui.Run(ui.Dependencies{
		Config:           rt.Config,
		ChatStore:        rt.ChatStore,
		NodeStore:        rt.NodeStore,
		Bus:              rt.Bus,
		LastSelectedChat: rt.Config.UI.LastSelectedChat,
		Sender:           rt.Radio,
		LocalNodeID:      rt.Radio.LocalNodeID,
		OnSave:           rt.SaveAndApplyConfig,
		OnChatSelected:   rt.RememberSelectedChat,
		OnClearDB:        rt.ClearDatabase,
		OnQuit: func() {
			stop()
			closeRuntime()
		},
	})
	if err != nil {
		return fmt.Errorf("run ui: %w", err)
	}

	return nil
}
