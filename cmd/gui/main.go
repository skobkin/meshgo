package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"io"
	"log/slog"
	"os"
	"os/signal"
	"strings"
	"sync"
	"syscall"

	"fyne.io/fyne/v2"
	fyneapp "fyne.io/fyne/v2/app"
	"fyne.io/fyne/v2/dialog"

	"github.com/skobkin/meshgo/internal/app"
	"github.com/skobkin/meshgo/internal/platform"
	"github.com/skobkin/meshgo/internal/ui"
)

const alreadyRunningMessage = "meshgo is already running for this user.\nClose the existing instance before starting another copy."

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

	slog.Info("acquiring single-instance lock", "app_id", app.Name)
	instanceLock, err := platform.AcquireInstanceLock(app.Name)
	if err != nil {
		if errors.Is(err, platform.ErrInstanceAlreadyRunning) {
			slog.Warn("single-instance lock contention: another app instance is already running", "app_id", app.Name)
			showAlreadyRunningDialog(alreadyRunningMessage)

			return fmt.Errorf("acquire instance lock: %w", err)
		}
		if errors.Is(err, platform.ErrInstanceLockUnsupported) {
			slog.Warn("single-instance lock is not supported on this platform; continuing without lock", "error", err)
		} else {
			return fmt.Errorf("acquire instance lock: %w", err)
		}
	} else {
		slog.Info("single-instance lock acquired", "app_id", app.Name)
		defer func() {
			if err := instanceLock.Release(); err != nil {
				slog.Warn("release instance lock", "error", err)
			}
		}()

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

	uiDeps := ui.BuildRuntimeDependencies(rt, ui.LaunchOptions{StartHidden: opts.StartHidden}, func() {
		stop()
		closeRuntime()
	})

	err = ui.Run(uiDeps)
	if err != nil {
		return fmt.Errorf("run ui: %w", err)
	}

	return nil
}

func showAlreadyRunningDialog(message string) {
	_, _ = fmt.Fprintln(os.Stderr, message)

	defer func() {
		if recovered := recover(); recovered != nil {
			slog.Warn("show already-running dialog", "error", recovered)
		}
	}()

	fyApp := fyneapp.New()
	window := fyApp.NewWindow("meshgo")
	window.Resize(fyne.NewSize(500, 160))

	var quitOnce sync.Once
	quit := func() {
		quitOnce.Do(func() {
			fyApp.Quit()
		})
	}

	info := dialog.NewInformation("meshgo is already running", message, window)
	info.SetOnClosed(quit)
	window.SetCloseIntercept(quit)
	window.Show()
	info.Show()
	fyApp.Run()
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
