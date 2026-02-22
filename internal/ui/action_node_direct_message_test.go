package ui

import (
	"testing"

	"github.com/skobkin/meshgo/internal/domain"
)

func TestHandleNodeDirectMessageAction_CreatesDMChatAndRequestsOpen(t *testing.T) {
	store := domain.NewChatStore()
	dep := RuntimeDependencies{
		Data: DataDependencies{
			ChatStore: store,
		},
	}

	switchCalled := false
	requestedChatKey := ""
	handleNodeDirectMessageAction(
		dep,
		func() {
			switchCalled = true
		},
		func(chatKey string) {
			requestedChatKey = chatKey
		},
		domain.Node{NodeID: "!1234abcd"},
	)

	if !switchCalled {
		t.Fatalf("expected chat tab switch callback to be called")
	}
	if requestedChatKey != "dm:!1234abcd" {
		t.Fatalf("unexpected requested chat key: %q", requestedChatKey)
	}
	chat, ok := store.ChatByKey("dm:!1234abcd")
	if !ok {
		t.Fatalf("expected dm chat to be upserted")
	}
	if chat.Type != domain.ChatTypeDM {
		t.Fatalf("unexpected chat type: %v", chat.Type)
	}
	if chat.Title != "dm:!1234abcd" {
		t.Fatalf("unexpected chat title: %q", chat.Title)
	}
}

func TestHandleNodeDirectMessageAction_InvalidNodeIDSkipsAction(t *testing.T) {
	store := domain.NewChatStore()
	dep := RuntimeDependencies{
		Data: DataDependencies{
			ChatStore: store,
		},
	}

	switchCalled := false
	requestCalled := false
	handleNodeDirectMessageAction(
		dep,
		func() {
			switchCalled = true
		},
		func(chatKey string) {
			requestCalled = true
		},
		domain.Node{NodeID: "unknown"},
	)

	if switchCalled {
		t.Fatalf("chat tab switch callback should not be called for invalid node")
	}
	if requestCalled {
		t.Fatalf("chat open callback should not be called for invalid node")
	}
	if _, ok := store.ChatByKey("dm:unknown"); ok {
		t.Fatalf("invalid node should not create a chat")
	}
}

func TestHandleNodeDirectMessageAction_NilChatStoreSkipsAction(t *testing.T) {
	dep := RuntimeDependencies{}

	switchCalled := false
	requestCalled := false
	handleNodeDirectMessageAction(
		dep,
		func() {
			switchCalled = true
		},
		func(chatKey string) {
			requestCalled = true
		},
		domain.Node{NodeID: "!1234abcd"},
	)

	if switchCalled {
		t.Fatalf("chat tab switch callback should not be called without chat store")
	}
	if requestCalled {
		t.Fatalf("chat open callback should not be called without chat store")
	}
}
