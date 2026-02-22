package ui

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"fyne.io/fyne/v2"
	fynetest "fyne.io/fyne/v2/test"
	"github.com/skobkin/meshgo/internal/app"
	"github.com/skobkin/meshgo/internal/bus"
	"github.com/skobkin/meshgo/internal/domain"
	"github.com/skobkin/meshgo/internal/radio/busmsg"
)

type tracerouteActionSpy struct {
	startCalls int
	lastTarget app.TracerouteTarget
	initial    busmsg.TracerouteUpdate
	err        error
}

func (s *tracerouteActionSpy) StartTraceroute(_ context.Context, target app.TracerouteTarget) (busmsg.TracerouteUpdate, error) {
	s.startCalls++
	s.lastTarget = target

	return s.initial, s.err
}

func TestHandleNodeTracerouteAction_NilWindowSkipsAction(t *testing.T) {
	spy := &tracerouteActionSpy{}

	handleNodeTracerouteAction(nil, RuntimeDependencies{
		Actions: ActionDependencies{Traceroute: spy},
	}, domain.Node{NodeID: "!0000002a"})

	if spy.startCalls != 0 {
		t.Fatalf("expected traceroute action to be skipped for nil window")
	}
}

func TestHandleNodeTracerouteAction_MissingTracerouteActionShowsError(t *testing.T) {
	fyApp := fynetest.NewApp()
	t.Cleanup(fyApp.Quit)
	window := fyApp.NewWindow("test")

	var gotErr error
	origShowError := tracerouteShowErrorDialog
	tracerouteShowErrorDialog = func(err error, _ fyne.Window) {
		gotErr = err
	}
	t.Cleanup(func() {
		tracerouteShowErrorDialog = origShowError
	})

	handleNodeTracerouteAction(window, RuntimeDependencies{}, domain.Node{NodeID: "!0000002a"})

	if gotErr == nil {
		t.Fatalf("expected missing action error")
	}
	if !strings.Contains(gotErr.Error(), "traceroute is unavailable") {
		t.Fatalf("unexpected error text: %q", gotErr.Error())
	}
}

func TestHandleNodeTracerouteAction_CooldownErrorShowsRemainingTime(t *testing.T) {
	fyApp := fynetest.NewApp()
	t.Cleanup(fyApp.Quit)
	window := fyApp.NewWindow("test")

	spy := &tracerouteActionSpy{
		err: &app.TracerouteCooldownError{Remaining: 500 * time.Millisecond},
	}

	var gotErr error
	origShowError := tracerouteShowErrorDialog
	tracerouteShowErrorDialog = func(err error, _ fyne.Window) {
		gotErr = err
	}
	t.Cleanup(func() {
		tracerouteShowErrorDialog = origShowError
	})

	handleNodeTracerouteAction(window, RuntimeDependencies{
		Actions: ActionDependencies{Traceroute: spy},
	}, domain.Node{NodeID: "!0000002a"})

	if gotErr == nil {
		t.Fatalf("expected cooldown error to be shown")
	}
	if !strings.Contains(gotErr.Error(), "try again in 1s") {
		t.Fatalf("unexpected cooldown text: %q", gotErr.Error())
	}
}

func TestHandleNodeTracerouteAction_StartErrorShowsOriginalError(t *testing.T) {
	fyApp := fynetest.NewApp()
	t.Cleanup(fyApp.Quit)
	window := fyApp.NewWindow("test")

	startErr := errors.New("boom")
	spy := &tracerouteActionSpy{err: startErr}

	var gotErr error
	origShowError := tracerouteShowErrorDialog
	tracerouteShowErrorDialog = func(err error, _ fyne.Window) {
		gotErr = err
	}
	t.Cleanup(func() {
		tracerouteShowErrorDialog = origShowError
	})

	handleNodeTracerouteAction(window, RuntimeDependencies{
		Actions: ActionDependencies{Traceroute: spy},
	}, domain.Node{NodeID: "!0000002a"})

	if !errors.Is(gotErr, startErr) {
		t.Fatalf("expected original start error, got %v", gotErr)
	}
}

func TestHandleNodeTracerouteAction_SuccessShowsModal(t *testing.T) {
	fyApp := fynetest.NewApp()
	t.Cleanup(fyApp.Quit)
	window := fyApp.NewWindow("test")

	initial := busmsg.TracerouteUpdate{RequestID: 1}
	spy := &tracerouteActionSpy{initial: initial}
	node := domain.Node{NodeID: "!0000002a"}
	nodeStore := domain.NewNodeStore()

	modalCalled := 0
	origShowModal := tracerouteShowModal
	tracerouteShowModal = func(
		gotWindow fyne.Window,
		_ bus.MessageBus,
		gotNodeStore *domain.NodeStore,
		gotNode domain.Node,
		gotInitial busmsg.TracerouteUpdate,
	) {
		modalCalled++
		if gotWindow != window {
			t.Fatalf("unexpected window passed to modal")
		}
		if gotNodeStore != nodeStore {
			t.Fatalf("unexpected node store passed to modal")
		}
		if gotNode.NodeID != node.NodeID {
			t.Fatalf("unexpected node passed to modal: %q", gotNode.NodeID)
		}
		if gotInitial.RequestID != initial.RequestID {
			t.Fatalf("unexpected initial update passed to modal: %d", gotInitial.RequestID)
		}
	}
	t.Cleanup(func() {
		tracerouteShowModal = origShowModal
	})

	handleNodeTracerouteAction(window, RuntimeDependencies{
		Data: DataDependencies{
			NodeStore: nodeStore,
		},
		Actions: ActionDependencies{
			Traceroute: spy,
		},
	}, node)

	if spy.startCalls != 1 {
		t.Fatalf("expected exactly one traceroute start call, got %d", spy.startCalls)
	}
	if spy.lastTarget.NodeID != node.NodeID {
		t.Fatalf("unexpected traceroute target node id: %q", spy.lastTarget.NodeID)
	}
	if modalCalled != 1 {
		t.Fatalf("expected modal to be shown once, got %d", modalCalled)
	}
}
