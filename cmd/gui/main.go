package main

import (
	"context"
	"flag"
	"fmt"
	"io"
	"log/slog"
	"os"
	"os/signal"
	"strings"
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
	opts, err := parseLaunchOptions(os.Args[1:])
	if err != nil {
		return fmt.Errorf("parse launch options: %w", err)
	}

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

	deps := ui.NewDependenciesFromRuntime(rt, ui.LaunchOptions{StartHidden: opts.StartHidden}, func() {
		stop()
		closeRuntime()
	})

	err = ui.Run(deps)
	if err != nil {
		return fmt.Errorf("run ui: %w", err)
	}

	return nil
}

type launchOptions struct {
	StartHidden bool
}

func parseLaunchOptions(args []string) (launchOptions, error) {
	fs := flag.NewFlagSet("meshgo", flag.ContinueOnError)
	fs.SetOutput(io.Discard)

	startHidden := fs.Bool("start-hidden", false, "start app with hidden window")
	if err := fs.Parse(args); err != nil {
		return launchOptions{}, err
	}
	if fs.NArg() > 0 {
		return launchOptions{}, fmt.Errorf("unexpected positional arguments: %s", strings.Join(fs.Args(), ", "))
	}

	return launchOptions{StartHidden: *startHidden}, nil
}
