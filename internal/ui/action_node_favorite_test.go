package ui

import (
	"context"
	"errors"
	"strings"
	"sync"
	"testing"
	"time"

	"fyne.io/fyne/v2"
	fynetest "fyne.io/fyne/v2/test"
	"github.com/skobkin/meshgo/internal/domain"
)

type nodeFavoriteActionSpy struct {
	mu       sync.Mutex
	calls    int
	waitFor  time.Duration
	nodeID   string
	favorite bool
	err      error
}

func (s *nodeFavoriteActionSpy) SetFavorite(_ context.Context, targetNodeID string, favorite bool) error {
	if s.waitFor > 0 {
		time.Sleep(s.waitFor)
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	s.calls++
	s.nodeID = targetNodeID
	s.favorite = favorite

	return s.err
}

func (s *nodeFavoriteActionSpy) snapshot() (calls int, nodeID string, favorite bool) {
	s.mu.Lock()
	defer s.mu.Unlock()

	return s.calls, s.nodeID, s.favorite
}

func TestHandleNodeFavoriteAction_NilWindowSkipsAction(t *testing.T) {
	spy := &nodeFavoriteActionSpy{}
	handleNodeFavoriteAction(nil, RuntimeDependencies{
		Actions: ActionDependencies{NodeFavorite: spy},
	}, domain.Node{NodeID: "!0000002a"}, true)

	_, nodeID, _ := spy.snapshot()
	if nodeID != "" {
		t.Fatalf("expected no action call for nil window")
	}
}

func TestHandleNodeFavoriteAction_MissingActionShowsError(t *testing.T) {
	fyApp := fynetest.NewApp()
	t.Cleanup(fyApp.Quit)
	window := fyApp.NewWindow("test")

	var gotErr error
	origShowError := nodeFavoriteShowErrorDialog
	nodeFavoriteShowErrorDialog = func(err error, _ fyne.Window) {
		gotErr = err
	}
	t.Cleanup(func() {
		nodeFavoriteShowErrorDialog = origShowError
	})

	handleNodeFavoriteAction(window, RuntimeDependencies{}, domain.Node{NodeID: "!0000002a"}, true)

	if gotErr == nil {
		t.Fatalf("expected missing action error")
	}
	if !strings.Contains(gotErr.Error(), "favorite action is unavailable") {
		t.Fatalf("unexpected error text: %q", gotErr.Error())
	}
}

func TestHandleNodeFavoriteAction_PassesNodeAndState(t *testing.T) {
	fyApp := fynetest.NewApp()
	t.Cleanup(fyApp.Quit)
	window := fyApp.NewWindow("test")
	spy := &nodeFavoriteActionSpy{}

	handleNodeFavoriteAction(window, RuntimeDependencies{
		Actions: ActionDependencies{NodeFavorite: spy},
	}, domain.Node{NodeID: "!0000002a"}, false)

	waitForCondition(t, func() bool {
		calls, _, _ := spy.snapshot()

		return calls == 1
	})
	_, nodeID, favorite := spy.snapshot()
	if nodeID != "!0000002a" {
		t.Fatalf("unexpected node id: %q", nodeID)
	}
	if favorite {
		t.Fatalf("expected favorite=false")
	}
}

func TestHandleNodeFavoriteAction_ErrorIsShown(t *testing.T) {
	fyApp := fynetest.NewApp()
	t.Cleanup(fyApp.Quit)
	window := fyApp.NewWindow("test")
	wantErr := errors.New("boom")
	spy := &nodeFavoriteActionSpy{err: wantErr}

	var gotErr error
	origShowError := nodeFavoriteShowErrorDialog
	nodeFavoriteShowErrorDialog = func(err error, _ fyne.Window) {
		gotErr = err
	}
	t.Cleanup(func() {
		nodeFavoriteShowErrorDialog = origShowError
	})

	handleNodeFavoriteAction(window, RuntimeDependencies{
		Actions: ActionDependencies{NodeFavorite: spy},
	}, domain.Node{NodeID: "!0000002a"}, true)

	waitForCondition(t, func() bool { return gotErr != nil })
	if !errors.Is(gotErr, wantErr) {
		t.Fatalf("expected propagated error, got %v", gotErr)
	}
}
