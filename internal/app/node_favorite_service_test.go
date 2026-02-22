package app

import (
	"context"
	"errors"
	"io"
	"log/slog"
	"testing"

	meshbus "github.com/skobkin/meshgo/internal/bus"
	"github.com/skobkin/meshgo/internal/domain"
	"github.com/skobkin/meshgo/internal/radio/busmsg"
	generated "github.com/skobkin/meshgo/internal/radio/meshtasticpb"
)

type nodeFavoriteRadioSpy struct {
	to           uint32
	channel      uint32
	wantResponse bool
	payload      *generated.AdminMessage
	err          error
}

func (s *nodeFavoriteRadioSpy) SendAdmin(to uint32, channel uint32, wantResponse bool, payload *generated.AdminMessage) (string, error) {
	s.to = to
	s.channel = channel
	s.wantResponse = wantResponse
	s.payload = payload
	if s.err != nil {
		return "", s.err
	}

	return "42", nil
}

func TestNodeFavoriteServiceSetFavorite(t *testing.T) {
	radio := &nodeFavoriteRadioSpy{}
	store := domain.NewNodeStore()
	store.Upsert(domain.Node{NodeID: "!0000002a"})

	messageBus := meshbus.New(discardLogger())
	sub := messageBus.Subscribe(meshbus.TopicNodeCore)
	defer messageBus.Unsubscribe(sub, meshbus.TopicNodeCore)

	service := NewNodeFavoriteService(
		radio,
		store,
		messageBus,
		func() string { return "!00000001" },
		func() (busmsg.ConnectionStatus, bool) {
			return busmsg.ConnectionStatus{State: busmsg.ConnectionStateConnected}, true
		},
		discardLogger(),
	)

	if err := service.SetFavorite(context.Background(), "!0000002a", true); err != nil {
		t.Fatalf("set favorite: %v", err)
	}
	if radio.to != 0x1 {
		t.Fatalf("unexpected admin target node: %d", radio.to)
	}
	if radio.channel != nodeFavoriteAdminChannel {
		t.Fatalf("unexpected admin channel: %d", radio.channel)
	}
	if radio.wantResponse {
		t.Fatalf("favorite admin should not request direct response")
	}
	if radio.payload == nil || radio.payload.GetSetFavoriteNode() != 0x2a {
		t.Fatalf("expected set_favorite_node payload, got %+v", radio.payload)
	}

	raw := <-sub
	update, ok := raw.(domain.NodeCoreUpdate)
	if !ok {
		t.Fatalf("expected NodeCoreUpdate event, got %T", raw)
	}
	if update.Core.NodeID != "!0000002a" {
		t.Fatalf("unexpected update node id: %q", update.Core.NodeID)
	}
	if update.Core.IsFavorite == nil || !*update.Core.IsFavorite {
		t.Fatalf("expected favorite=true in update, got %v", update.Core.IsFavorite)
	}
}

func TestNodeFavoriteServiceUnsetFavoriteWithoutBusUpdatesStore(t *testing.T) {
	radio := &nodeFavoriteRadioSpy{}
	store := domain.NewNodeStore()
	initial := true
	store.Upsert(domain.Node{NodeID: "!0000002a", IsFavorite: &initial})

	service := NewNodeFavoriteService(
		radio,
		store,
		nil,
		func() string { return "!00000001" },
		func() (busmsg.ConnectionStatus, bool) {
			return busmsg.ConnectionStatus{State: busmsg.ConnectionStateConnected}, true
		},
		discardLogger(),
	)

	if err := service.SetFavorite(context.Background(), "!0000002a", false); err != nil {
		t.Fatalf("unset favorite: %v", err)
	}
	if radio.payload == nil || radio.payload.GetRemoveFavoriteNode() != 0x2a {
		t.Fatalf("expected remove_favorite_node payload, got %+v", radio.payload)
	}

	node, ok := store.Get("!0000002a")
	if !ok {
		t.Fatalf("expected node in store")
	}
	if node.IsFavorite == nil || *node.IsFavorite {
		t.Fatalf("expected favorite=false in store, got %v", node.IsFavorite)
	}
}

func TestNodeFavoriteServiceSetFavoriteValidation(t *testing.T) {
	t.Run("fails when disconnected", func(t *testing.T) {
		service := NewNodeFavoriteService(
			&nodeFavoriteRadioSpy{},
			domain.NewNodeStore(),
			nil,
			func() string { return "!00000001" },
			func() (busmsg.ConnectionStatus, bool) {
				return busmsg.ConnectionStatus{State: busmsg.ConnectionStateDisconnected}, true
			},
			discardLogger(),
		)
		if err := service.SetFavorite(context.Background(), "!0000002a", true); err == nil {
			t.Fatalf("expected disconnected error")
		}
	})

	t.Run("propagates radio error", func(t *testing.T) {
		wantErr := errors.New("boom")
		service := NewNodeFavoriteService(
			&nodeFavoriteRadioSpy{err: wantErr},
			domain.NewNodeStore(),
			nil,
			func() string { return "!00000001" },
			func() (busmsg.ConnectionStatus, bool) {
				return busmsg.ConnectionStatus{State: busmsg.ConnectionStateConnected}, true
			},
			discardLogger(),
		)
		err := service.SetFavorite(context.Background(), "!0000002a", true)
		if !errors.Is(err, wantErr) {
			t.Fatalf("expected radio error, got %v", err)
		}
	})
}

func discardLogger() *slog.Logger {
	return slog.New(slog.NewTextHandler(io.Discard, nil))
}
