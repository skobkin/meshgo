package storage

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"time"

	"meshgo/internal/core"
)

func setupTestDB(t *testing.T) (*SQLiteStore, func()) {
	tmpDir, err := os.MkdirTemp("", "meshgo_test_*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}

	store, err := NewSQLiteStore(tmpDir)
	if err != nil {
		os.RemoveAll(tmpDir)
		t.Fatalf("Failed to create test store: %v", err)
	}

	cleanup := func() {
		store.Close()
		os.RemoveAll(tmpDir)
	}

	return store, cleanup
}

func TestNewSQLiteStore(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "meshgo_test_*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	store, err := NewSQLiteStore(tmpDir)
	if err != nil {
		t.Fatalf("NewSQLiteStore failed: %v", err)
	}
	defer store.Close()

	// Check database file was created
	dbPath := filepath.Join(tmpDir, "meshgo.db")
	if _, err := os.Stat(dbPath); os.IsNotExist(err) {
		t.Error("Database file was not created")
	}
}

func TestSQLiteStore_SaveAndGetMessage(t *testing.T) {
	store, cleanup := setupTestDB(t)
	defer cleanup()

	ctx := context.Background()
	msg := &core.Message{
		ChatID:    "test_chat",
		SenderID:  "node_123",
		PortNum:   1,
		Text:      "Hello, World!",
		Timestamp: time.Now().UTC().Truncate(time.Second),
		IsUnread:  true,
	}

	// Save message
	err := store.SaveMessage(ctx, msg)
	if err != nil {
		t.Fatalf("SaveMessage failed: %v", err)
	}

	// Get messages
	messages, err := store.GetMessages(ctx, "test_chat", 10, 0)
	if err != nil {
		t.Fatalf("GetMessages failed: %v", err)
	}

	if len(messages) != 1 {
		t.Fatalf("Expected 1 message, got %d", len(messages))
	}

	retrieved := messages[0]
	if retrieved.ChatID != msg.ChatID {
		t.Errorf("ChatID mismatch: got %s, want %s", retrieved.ChatID, msg.ChatID)
	}
	if retrieved.SenderID != msg.SenderID {
		t.Errorf("SenderID mismatch: got %s, want %s", retrieved.SenderID, msg.SenderID)
	}
	if retrieved.Text != msg.Text {
		t.Errorf("Text mismatch: got %s, want %s", retrieved.Text, msg.Text)
	}
	if retrieved.PortNum != msg.PortNum {
		t.Errorf("PortNum mismatch: got %d, want %d", retrieved.PortNum, msg.PortNum)
	}
	if !retrieved.IsUnread {
		t.Error("Message should be unread initially")
	}
}

func TestSQLiteStore_UnreadCount(t *testing.T) {
	store, cleanup := setupTestDB(t)
	defer cleanup()

	ctx := context.Background()

	// Save multiple messages
	messages := []*core.Message{
		{ChatID: "chat1", SenderID: "node1", Text: "Message 1", IsUnread: true, Timestamp: time.Now()},
		{ChatID: "chat1", SenderID: "node2", Text: "Message 2", IsUnread: true, Timestamp: time.Now()},
		{ChatID: "chat2", SenderID: "node1", Text: "Message 3", IsUnread: true, Timestamp: time.Now()},
		{ChatID: "chat1", SenderID: "node1", Text: "Message 4", IsUnread: false, Timestamp: time.Now()}, // Read message
	}

	for _, msg := range messages {
		if err := store.SaveMessage(ctx, msg); err != nil {
			t.Fatalf("SaveMessage failed: %v", err)
		}
	}

	// Test unread count for specific chat
	count, err := store.GetUnreadCount(ctx, "chat1")
	if err != nil {
		t.Fatalf("GetUnreadCount failed: %v", err)
	}
	if count != 2 {
		t.Errorf("Expected 2 unread messages in chat1, got %d", count)
	}

	// Test total unread count
	totalCount, err := store.GetTotalUnreadCount(ctx)
	if err != nil {
		t.Fatalf("GetTotalUnreadCount failed: %v", err)
	}
	if totalCount != 3 {
		t.Errorf("Expected 3 total unread messages, got %d", totalCount)
	}
}

func TestSQLiteStore_MarkAsRead(t *testing.T) {
	store, cleanup := setupTestDB(t)
	defer cleanup()

	ctx := context.Background()

	// Save unread messages
	messages := []*core.Message{
		{ChatID: "chat1", SenderID: "node1", Text: "Message 1", IsUnread: true, Timestamp: time.Now()},
		{ChatID: "chat1", SenderID: "node2", Text: "Message 2", IsUnread: true, Timestamp: time.Now()},
	}

	for _, msg := range messages {
		if err := store.SaveMessage(ctx, msg); err != nil {
			t.Fatalf("SaveMessage failed: %v", err)
		}
	}

	// Mark as read
	err := store.MarkAsRead(ctx, "chat1")
	if err != nil {
		t.Fatalf("MarkAsRead failed: %v", err)
	}

	// Verify unread count is now 0
	count, err := store.GetUnreadCount(ctx, "chat1")
	if err != nil {
		t.Fatalf("GetUnreadCount failed: %v", err)
	}
	if count != 0 {
		t.Errorf("Expected 0 unread messages after marking as read, got %d", count)
	}
}

func TestSQLiteStore_SaveAndGetNode(t *testing.T) {
	store, cleanup := setupTestDB(t)
	defer cleanup()

	ctx := context.Background()
	node := &core.Node{
		ID:            "node_123",
		ShortName:     "TestNode",
		LongName:      "Test Node Device",
		RSSI:          -85,
		SNR:           12.5,
		SignalQuality: int(core.SignalGood),
		LastHeard:     time.Now().UTC().Truncate(time.Second),
		Favorite:      true,
		Ignored:       false,
		Position: &core.Position{
			LatitudeI:  375594120,
			LongitudeI: -1213894470,
			Altitude:   100,
			Time:       uint32(time.Now().Unix()),
		},
		DeviceMetrics: &core.DeviceMetrics{
			BatteryLevel: 85,
			Voltage:      3.7,
		},
	}

	// Save node
	err := store.SaveNode(ctx, node)
	if err != nil {
		t.Fatalf("SaveNode failed: %v", err)
	}

	// Get node
	retrieved, err := store.GetNode(ctx, "node_123")
	if err != nil {
		t.Fatalf("GetNode failed: %v", err)
	}

	if retrieved == nil {
		t.Fatal("Retrieved node is nil")
	}

	if retrieved.ID != node.ID {
		t.Errorf("ID mismatch: got %s, want %s", retrieved.ID, node.ID)
	}
	if retrieved.ShortName != node.ShortName {
		t.Errorf("ShortName mismatch: got %s, want %s", retrieved.ShortName, node.ShortName)
	}
	if retrieved.RSSI != node.RSSI {
		t.Errorf("RSSI mismatch: got %d, want %d", retrieved.RSSI, node.RSSI)
	}
	if retrieved.SNR != node.SNR {
		t.Errorf("SNR mismatch: got %f, want %f", retrieved.SNR, node.SNR)
	}
	if retrieved.Favorite != node.Favorite {
		t.Errorf("Favorite mismatch: got %v, want %v", retrieved.Favorite, node.Favorite)
	}
	if retrieved.Position.LatitudeI != node.Position.LatitudeI {
		t.Errorf("Position LatitudeI mismatch: got %d, want %d", retrieved.Position.LatitudeI, node.Position.LatitudeI)
	}
	if retrieved.DeviceMetrics.BatteryLevel != node.DeviceMetrics.BatteryLevel {
		t.Errorf("BatteryLevel mismatch: got %d, want %d", retrieved.DeviceMetrics.BatteryLevel, node.DeviceMetrics.BatteryLevel)
	}
}

func TestSQLiteStore_GetAllNodes(t *testing.T) {
	store, cleanup := setupTestDB(t)
	defer cleanup()

	ctx := context.Background()

	// Save multiple nodes
	nodes := []*core.Node{
		{ID: "node1", ShortName: "Node1", LastHeard: time.Now()},
		{ID: "node2", ShortName: "Node2", LastHeard: time.Now()},
		{ID: "node3", ShortName: "Node3", LastHeard: time.Now()},
	}

	for _, node := range nodes {
		if err := store.SaveNode(ctx, node); err != nil {
			t.Fatalf("SaveNode failed: %v", err)
		}
	}

	// Get all nodes
	retrieved, err := store.GetAllNodes(ctx)
	if err != nil {
		t.Fatalf("GetAllNodes failed: %v", err)
	}

	if len(retrieved) != 3 {
		t.Errorf("Expected 3 nodes, got %d", len(retrieved))
	}

	// Check that all nodes were retrieved
	nodeIDs := make(map[string]bool)
	for _, node := range retrieved {
		nodeIDs[node.ID] = true
	}

	for _, expectedNode := range nodes {
		if !nodeIDs[expectedNode.ID] {
			t.Errorf("Node %s not found in retrieved nodes", expectedNode.ID)
		}
	}
}

func TestSQLiteStore_UpdateNodeFavorite(t *testing.T) {
	store, cleanup := setupTestDB(t)
	defer cleanup()

	ctx := context.Background()
	node := &core.Node{
		ID:        "node_123",
		ShortName: "TestNode",
		Favorite:  false,
		LastHeard: time.Now(),
	}

	// Save node
	err := store.SaveNode(ctx, node)
	if err != nil {
		t.Fatalf("SaveNode failed: %v", err)
	}

	// Update favorite status
	err = store.UpdateNodeFavorite(ctx, "node_123", true)
	if err != nil {
		t.Fatalf("UpdateNodeFavorite failed: %v", err)
	}

	// Verify update
	retrieved, err := store.GetNode(ctx, "node_123")
	if err != nil {
		t.Fatalf("GetNode failed: %v", err)
	}

	if !retrieved.Favorite {
		t.Error("Node should be marked as favorite")
	}
}

func TestSQLiteStore_DeleteNode(t *testing.T) {
	store, cleanup := setupTestDB(t)
	defer cleanup()

	ctx := context.Background()
	node := &core.Node{
		ID:        "node_123",
		ShortName: "TestNode",
		LastHeard: time.Now(),
	}

	// Save node
	err := store.SaveNode(ctx, node)
	if err != nil {
		t.Fatalf("SaveNode failed: %v", err)
	}

	// Delete node
	err = store.DeleteNode(ctx, "node_123")
	if err != nil {
		t.Fatalf("DeleteNode failed: %v", err)
	}

	// Verify deletion
	retrieved, err := store.GetNode(ctx, "node_123")
	if err != nil {
		t.Fatalf("GetNode failed: %v", err)
	}

	if retrieved != nil {
		t.Error("Node should have been deleted")
	}
}

func TestSQLiteStore_KeyValueOperations(t *testing.T) {
	store, cleanup := setupTestDB(t)
	defer cleanup()

	// Test string operations
	err := store.Set("test_key", "test_value")
	if err != nil {
		t.Fatalf("Set failed: %v", err)
	}

	value, err := store.Get("test_key")
	if err != nil {
		t.Fatalf("Get failed: %v", err)
	}

	if value != "test_value" {
		t.Errorf("Expected 'test_value', got '%s'", value)
	}

	// Test non-existent key (should return empty string, no error)
	value, err = store.Get("non_existent")
	if err != nil {
		t.Errorf("Get should not return error for non-existent key: %v", err)
	}
	if value != "" {
		t.Errorf("Expected empty string for non-existent key, got '%s'", value)
	}

	// Test boolean operations
	err = store.SetBool("bool_key", true)
	if err != nil {
		t.Fatalf("SetBool failed: %v", err)
	}

	boolValue := store.GetBool("bool_key", false)
	if !boolValue {
		t.Error("Expected true, got false")
	}

	// Test default value for non-existent bool
	defaultBool := store.GetBool("non_existent_bool", true)
	if !defaultBool {
		t.Error("Expected default value true")
	}

	// Test integer operations
	err = store.SetInt("int_key", 42)
	if err != nil {
		t.Fatalf("SetInt failed: %v", err)
	}

	intValue := store.GetInt("int_key", 0)
	if intValue != 42 {
		t.Errorf("Expected 42, got %d", intValue)
	}

	// Test default value for non-existent int
	defaultInt := store.GetInt("non_existent_int", 100)
	if defaultInt != 100 {
		t.Errorf("Expected default value 100, got %d", defaultInt)
	}
}

func TestSQLiteStore_GetAllChats(t *testing.T) {
	store, cleanup := setupTestDB(t)
	defer cleanup()

	ctx := context.Background()

	// Save messages to create chats
	messages := []*core.Message{
		{ChatID: "chat1", SenderID: "node1", Text: "Message 1", IsUnread: true, Timestamp: time.Now()},
		{ChatID: "chat2", SenderID: "node2", Text: "Message 2", IsUnread: false, Timestamp: time.Now()},
		{ChatID: "chat1", SenderID: "node3", Text: "Message 3", IsUnread: true, Timestamp: time.Now()},
	}

	for _, msg := range messages {
		if err := store.SaveMessage(ctx, msg); err != nil {
			t.Fatalf("SaveMessage failed: %v", err)
		}
	}

	// Get all chats
	chats, err := store.GetAllChats(ctx)
	if err != nil {
		t.Fatalf("GetAllChats failed: %v", err)
	}

	if len(chats) != 2 {
		t.Errorf("Expected 2 chats, got %d", len(chats))
	}

	// Find chat1 and verify unread count
	var chat1 *core.Chat
	for _, chat := range chats {
		if chat.ID == "chat1" {
			chat1 = chat
			break
		}
	}

	if chat1 == nil {
		t.Fatal("chat1 not found")
	}

	if chat1.UnreadCount != 2 {
		t.Errorf("Expected 2 unread messages in chat1, got %d", chat1.UnreadCount)
	}
}

func TestSQLiteStore_ClearAllChats(t *testing.T) {
	store, cleanup := setupTestDB(t)
	defer cleanup()

	ctx := context.Background()

	// Save messages
	msg := &core.Message{
		ChatID:    "test_chat",
		SenderID:  "node_123",
		Text:      "Test message",
		Timestamp: time.Now(),
	}

	err := store.SaveMessage(ctx, msg)
	if err != nil {
		t.Fatalf("SaveMessage failed: %v", err)
	}

	// Clear all chats
	err = store.ClearAllChats(ctx)
	if err != nil {
		t.Fatalf("ClearAllChats failed: %v", err)
	}

	// Verify chats are cleared
	chats, err := store.GetAllChats(ctx)
	if err != nil {
		t.Fatalf("GetAllChats failed: %v", err)
	}

	if len(chats) != 0 {
		t.Errorf("Expected 0 chats after clearing, got %d", len(chats))
	}
}

func TestSQLiteStore_ClearAllNodes(t *testing.T) {
	store, cleanup := setupTestDB(t)
	defer cleanup()

	ctx := context.Background()

	// Save node
	node := &core.Node{
		ID:        "node_123",
		ShortName: "TestNode",
		LastHeard: time.Now(),
	}

	err := store.SaveNode(ctx, node)
	if err != nil {
		t.Fatalf("SaveNode failed: %v", err)
	}

	// Clear all nodes
	err = store.ClearAllNodes(ctx)
	if err != nil {
		t.Fatalf("ClearAllNodes failed: %v", err)
	}

	// Verify nodes are cleared
	nodes, err := store.GetAllNodes(ctx)
	if err != nil {
		t.Fatalf("GetAllNodes failed: %v", err)
	}

	if len(nodes) != 0 {
		t.Errorf("Expected 0 nodes after clearing, got %d", len(nodes))
	}
}

func TestSQLiteStore_Pagination(t *testing.T) {
	store, cleanup := setupTestDB(t)
	defer cleanup()

	ctx := context.Background()

	// Save multiple messages with timestamps
	baseTime := time.Now().UTC().Truncate(time.Second)
	for i := 0; i < 10; i++ {
		msg := &core.Message{
			ChatID:    "test_chat",
			SenderID:  "node_123",
			Text:      fmt.Sprintf("Message %d", i),
			Timestamp: baseTime.Add(time.Duration(i) * time.Second),
		}
		if err := store.SaveMessage(ctx, msg); err != nil {
			t.Fatalf("SaveMessage failed: %v", err)
		}
	}

	// Test pagination - first page
	messages, err := store.GetMessages(ctx, "test_chat", 3, 0)
	if err != nil {
		t.Fatalf("GetMessages failed: %v", err)
	}

	if len(messages) != 3 {
		t.Errorf("Expected 3 messages on first page, got %d", len(messages))
	}

	// Messages should be ordered by timestamp DESC (newest first)
	if messages[0].Text != "Message 9" {
		t.Errorf("Expected newest message first, got %s", messages[0].Text)
	}

	// Test pagination - second page
	messages, err = store.GetMessages(ctx, "test_chat", 3, 3)
	if err != nil {
		t.Fatalf("GetMessages failed: %v", err)
	}

	if len(messages) != 3 {
		t.Errorf("Expected 3 messages on second page, got %d", len(messages))
	}

	if messages[0].Text != "Message 6" {
		t.Errorf("Expected 'Message 6' first on second page, got %s", messages[0].Text)
	}
}
