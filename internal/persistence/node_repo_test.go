package persistence

import (
	"context"
	"database/sql"
	"path/filepath"
	"testing"
	"time"

	"github.com/skobkin/meshgo/internal/domain"
)

func TestNodeRepoUpsertAndList_RoundTripsCoordinates(t *testing.T) {
	ctx := context.Background()
	dbPath := filepath.Join(t.TempDir(), "app.db")

	db, err := Open(ctx, dbPath)
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	defer func() { _ = db.Close() }()

	repo := NewNodeRepo(db)
	lat := 37.7749
	lon := -122.4194
	alt := int32(123)
	now := time.Now().UTC()

	if err := repo.Upsert(ctx, domain.Node{
		NodeID:      "!abcd1234",
		LongName:    "Alpha",
		Latitude:    &lat,
		Longitude:   &lon,
		Altitude:    &alt,
		LastHeardAt: now,
		UpdatedAt:   now,
	}); err != nil {
		t.Fatalf("upsert with coordinates: %v", err)
	}
	if err := repo.Upsert(ctx, domain.Node{
		NodeID:      "!abcd1234",
		ShortName:   "ALPH",
		LastHeardAt: now.Add(time.Second),
		UpdatedAt:   now.Add(time.Second),
	}); err != nil {
		t.Fatalf("upsert sparse update: %v", err)
	}

	nodes, err := repo.ListSortedByLastHeard(ctx)
	if err != nil {
		t.Fatalf("list nodes: %v", err)
	}
	if len(nodes) != 1 {
		t.Fatalf("expected one node, got %d", len(nodes))
	}
	if nodes[0].Latitude == nil || *nodes[0].Latitude != lat {
		t.Fatalf("expected latitude to roundtrip, got %v", nodes[0].Latitude)
	}
	if nodes[0].Longitude == nil || *nodes[0].Longitude != lon {
		t.Fatalf("expected longitude to roundtrip, got %v", nodes[0].Longitude)
	}
	if nodes[0].Altitude == nil || *nodes[0].Altitude != alt {
		t.Fatalf("expected altitude to roundtrip, got %v", nodes[0].Altitude)
	}
}

func TestOpen_MigratesV4DatabaseToV8(t *testing.T) {
	ctx := context.Background()
	dbPath := filepath.Join(t.TempDir(), "app.db")

	db, err := sql.Open("sqlite", dbPath)
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}

	stmts := []string{
		`CREATE TABLE nodes (
			node_id TEXT PRIMARY KEY,
			long_name TEXT,
			short_name TEXT,
			battery_level INTEGER NULL,
			voltage REAL NULL,
			temperature REAL NULL,
			humidity REAL NULL,
			pressure REAL NULL,
			air_quality_index REAL NULL,
			power_voltage REAL NULL,
			power_current REAL NULL,
			board_model TEXT NULL,
			device_role TEXT NULL,
			is_unmessageable INTEGER NULL,
			last_heard_at INTEGER,
			rssi INTEGER NULL,
			snr REAL NULL,
			updated_at INTEGER NOT NULL
		);`,
		`CREATE INDEX nodes_last_heard_at_idx ON nodes(last_heard_at DESC);`,
		`CREATE TABLE messages (
			local_id INTEGER PRIMARY KEY AUTOINCREMENT,
			chat_key TEXT NOT NULL,
			device_message_id TEXT NULL,
			direction INTEGER NOT NULL,
			body TEXT NOT NULL,
			status INTEGER NOT NULL,
			at INTEGER NOT NULL,
			meta_json TEXT NULL
		);`,
		`PRAGMA user_version = 4;`,
	}
	for _, stmt := range stmts {
		if _, err := db.ExecContext(ctx, stmt); err != nil {
			_ = db.Close()
			t.Fatalf("seed v4 schema: %v", err)
		}
	}
	_ = db.Close()

	migrated, err := Open(ctx, dbPath)
	if err != nil {
		t.Fatalf("open migrated db: %v", err)
	}
	defer func() { _ = migrated.Close() }()

	var version int
	if err := migrated.QueryRowContext(ctx, `PRAGMA user_version;`).Scan(&version); err != nil {
		t.Fatalf("read user_version: %v", err)
	}
	if version != 8 {
		t.Fatalf("expected schema version 8, got %d", version)
	}

	columns := make(map[string]bool)
	rows, err := migrated.QueryContext(ctx, `PRAGMA table_info(nodes);`)
	if err != nil {
		t.Fatalf("read table info: %v", err)
	}
	defer func() { _ = rows.Close() }()

	for rows.Next() {
		var (
			cid       int
			name      string
			typ       string
			notNull   int
			defaultV  sql.NullString
			primaryID int
		)
		if err := rows.Scan(&cid, &name, &typ, &notNull, &defaultV, &primaryID); err != nil {
			t.Fatalf("scan table info: %v", err)
		}
		columns[name] = true
	}
	if err := rows.Err(); err != nil {
		t.Fatalf("iterate table info: %v", err)
	}
	if !columns["latitude"] {
		t.Fatalf("expected latitude column after migration")
	}
	if !columns["longitude"] {
		t.Fatalf("expected longitude column after migration")
	}
	if !columns["channel"] {
		t.Fatalf("expected channel column after migration")
	}
	if !columns["altitude"] {
		t.Fatalf("expected altitude column after migration")
	}
	messageColumns := make(map[string]bool)
	messageRows, err := migrated.QueryContext(ctx, `PRAGMA table_info(messages);`)
	if err != nil {
		t.Fatalf("read messages table info: %v", err)
	}
	defer func() { _ = messageRows.Close() }()

	for messageRows.Next() {
		var (
			cid       int
			name      string
			typ       string
			notNull   int
			defaultV  sql.NullString
			primaryID int
		)
		if err := messageRows.Scan(&cid, &name, &typ, &notNull, &defaultV, &primaryID); err != nil {
			t.Fatalf("scan messages table info: %v", err)
		}
		messageColumns[name] = true
	}
	if err := messageRows.Err(); err != nil {
		t.Fatalf("iterate messages table info: %v", err)
	}
	if !messageColumns["reply_to_device_message_id"] {
		t.Fatalf("expected reply_to_device_message_id column after migration")
	}
	if !messageColumns["emoji"] {
		t.Fatalf("expected emoji column after migration")
	}

	var traceroutesTable string
	if err := migrated.QueryRowContext(ctx, `
		SELECT name
		FROM sqlite_master
		WHERE type = 'table' AND name = 'traceroutes'
	`).Scan(&traceroutesTable); err != nil {
		t.Fatalf("expected traceroutes table after migration: %v", err)
	}
	if traceroutesTable != "traceroutes" {
		t.Fatalf("unexpected traceroutes table name: %q", traceroutesTable)
	}
}
