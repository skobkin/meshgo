package persistence

import (
	"bytes"
	"context"
	"database/sql"
	"path/filepath"
	"testing"
	"time"

	"github.com/skobkin/meshgo/internal/domain"
)

func TestNodeReposRoundTrip_CorePositionTelemetryAndIdentityHistory(t *testing.T) {
	ctx := context.Background()
	db, err := Open(ctx, filepath.Join(t.TempDir(), "app.db"))
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	defer func() { _ = db.Close() }()

	coreRepo := NewNodeCoreRepo(db)
	positionRepo := NewNodePositionRepo(db)
	telemetryRepo := NewNodeTelemetryRepo(db)
	identityRepo := NewNodeIdentityHistoryRepo(db)

	now := time.Now().UTC()
	nodeID := "!abcd1234"
	channel := uint32(2)
	pubKey := []byte{1, 2, 3, 4}
	if err := coreRepo.Upsert(ctx, domain.NodeCoreUpdate{
		Core: domain.NodeCore{
			NodeID:          nodeID,
			LongName:        "Alpha",
			ShortName:       "ALPH",
			PublicKey:       pubKey,
			Channel:         &channel,
			BoardModel:      "T-Echo",
			FirmwareVersion: "2.5.1",
			LastHeardAt:     now,
			UpdatedAt:       now,
		},
		FromPacket: true,
		Type:       domain.NodeUpdateTypeNodeInfoPacket,
	}, 50); err != nil {
		t.Fatalf("upsert node core: %v", err)
	}

	lat := 37.7749
	lon := -122.4194
	alt := int32(123)
	precision := uint32(15)
	if err := positionRepo.Upsert(ctx, domain.NodePositionUpdate{
		Position: domain.NodePosition{
			NodeID:                nodeID,
			Channel:               &channel,
			Latitude:              &lat,
			Longitude:             &lon,
			Altitude:              &alt,
			PositionPrecisionBits: &precision,
			PositionUpdatedAt:     now.Add(-2 * time.Minute),
			ObservedAt:            now,
			UpdatedAt:             now,
		},
		FromPacket: true,
		Type:       domain.NodeUpdateTypePositionPacket,
	}, 100); err != nil {
		t.Fatalf("upsert node position: %v", err)
	}

	battery := uint32(88)
	voltage := 4.2
	uptime := uint32(3600)
	channelUtil := 17.5
	airUtilTx := 2.3
	temp := 23.1
	if err := telemetryRepo.Upsert(ctx, domain.NodeTelemetryUpdate{
		Telemetry: domain.NodeTelemetry{
			NodeID:             nodeID,
			Channel:            &channel,
			BatteryLevel:       &battery,
			Voltage:            &voltage,
			UptimeSeconds:      &uptime,
			ChannelUtilization: &channelUtil,
			AirUtilTx:          &airUtilTx,
			Temperature:        &temp,
			ObservedAt:         now,
			UpdatedAt:          now,
		},
		FromPacket: true,
		Type:       domain.NodeUpdateTypeTelemetryPacket,
	}, 250); err != nil {
		t.Fatalf("upsert node telemetry: %v", err)
	}

	coreList, err := coreRepo.ListSortedByLastHeard(ctx)
	if err != nil {
		t.Fatalf("list node core: %v", err)
	}
	if len(coreList) != 1 {
		t.Fatalf("expected one core row, got %d", len(coreList))
	}
	if !bytes.Equal(coreList[0].PublicKey, pubKey) {
		t.Fatalf("expected public key to roundtrip, got %v", coreList[0].PublicKey)
	}

	positionList, err := positionRepo.ListLatest(ctx)
	if err != nil {
		t.Fatalf("list node position latest: %v", err)
	}
	if len(positionList) != 1 {
		t.Fatalf("expected one position latest row, got %d", len(positionList))
	}
	if positionList[0].Latitude == nil || *positionList[0].Latitude != lat {
		t.Fatalf("expected latitude to roundtrip, got %v", positionList[0].Latitude)
	}

	telemetryList, err := telemetryRepo.ListLatest(ctx)
	if err != nil {
		t.Fatalf("list node telemetry latest: %v", err)
	}
	if len(telemetryList) != 1 {
		t.Fatalf("expected one telemetry latest row, got %d", len(telemetryList))
	}
	if telemetryList[0].BatteryLevel == nil || *telemetryList[0].BatteryLevel != battery {
		t.Fatalf("expected battery to roundtrip, got %v", telemetryList[0].BatteryLevel)
	}

	history, err := identityRepo.ListHistoryByNodeID(ctx, domain.NodeHistoryQuery{
		NodeID: nodeID,
		Limit:  10,
		Order:  domain.HistorySortDescending,
	})
	if err != nil {
		t.Fatalf("list identity history: %v", err)
	}
	if len(history) != 1 {
		t.Fatalf("expected one identity history row, got %d", len(history))
	}

	// Repeating the same identity values should not add duplicate history entries.
	if err := coreRepo.Upsert(ctx, domain.NodeCoreUpdate{
		Core: domain.NodeCore{
			NodeID:      nodeID,
			LongName:    "Alpha",
			ShortName:   "ALPH",
			PublicKey:   pubKey,
			LastHeardAt: now.Add(time.Second),
			UpdatedAt:   now.Add(time.Second),
		},
		FromPacket: true,
		Type:       domain.NodeUpdateTypeNodeInfoPacket,
	}, 50); err != nil {
		t.Fatalf("repeat upsert node core: %v", err)
	}
	history, err = identityRepo.ListHistoryByNodeID(ctx, domain.NodeHistoryQuery{
		NodeID: nodeID,
		Limit:  10,
		Order:  domain.HistorySortDescending,
	})
	if err != nil {
		t.Fatalf("list identity history after duplicate upsert: %v", err)
	}
	if len(history) != 1 {
		t.Fatalf("expected duplicate identity update to be skipped, got %d rows", len(history))
	}
}

func TestOpenMigratesV11ToV12AndBackfillsSplitTables(t *testing.T) {
	ctx := context.Background()
	dbPath := filepath.Join(t.TempDir(), "app.db")

	db, err := sql.Open("sqlite", dbPath)
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}

	now := time.Now().UnixMilli()
	stmts := []string{
		`CREATE TABLE nodes (
			node_id TEXT PRIMARY KEY,
			long_name TEXT,
			short_name TEXT,
			public_key BLOB NULL,
			channel INTEGER NULL,
			latitude REAL NULL,
			longitude REAL NULL,
			altitude INTEGER NULL,
			precision_bits INTEGER NULL,
			battery_level INTEGER NULL,
			voltage REAL NULL,
			uptime_seconds INTEGER NULL,
			channel_utilization REAL NULL,
			air_util_tx REAL NULL,
			temperature REAL NULL,
			humidity REAL NULL,
			pressure REAL NULL,
			air_quality_index REAL NULL,
			power_voltage REAL NULL,
			power_current REAL NULL,
			board_model TEXT NULL,
			firmware_version TEXT NULL,
			device_role TEXT NULL,
			is_unmessageable INTEGER NULL,
			position_updated_at INTEGER NULL,
			last_heard_at INTEGER,
			rssi INTEGER NULL,
			snr REAL NULL,
			updated_at INTEGER NOT NULL
		);`,
		`CREATE INDEX nodes_last_heard_at_idx ON nodes(last_heard_at DESC);`,
		`INSERT INTO nodes(node_id, long_name, short_name, public_key, channel, latitude, longitude, altitude, precision_bits, battery_level, voltage, uptime_seconds, channel_utilization, air_util_tx, temperature, humidity, pressure, air_quality_index, power_voltage, power_current, board_model, firmware_version, device_role, is_unmessageable, position_updated_at, last_heard_at, rssi, snr, updated_at)
		 VALUES(?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		`PRAGMA user_version = 11;`,
	}
	for i, stmt := range stmts {
		if i == 2 {
			if _, err := db.ExecContext(ctx, stmt,
				"!00000001", "Alpha", "ALPH", []byte{9, 8, 7, 6}, 1,
				37.7, -122.4, 120, 14, 87, 4.2, 3600, 17.2, 2.1, 22.3, 44.1, 1002.3, 12.4, 5.1, 0.3,
				"T-Echo", "2.5.1", "ROUTER", 0, now-1000, now-500, -91, 6.2, now,
			); err != nil {
				_ = db.Close()
				t.Fatalf("seed v11 node row: %v", err)
			}

			continue
		}
		if _, err := db.ExecContext(ctx, stmt); err != nil {
			_ = db.Close()
			t.Fatalf("seed v11 schema statement %d: %v", i, err)
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
	if version != 12 {
		t.Fatalf("expected schema version 12, got %d", version)
	}

	if hasColumn(t, migrated, "nodes", "latitude") {
		t.Fatalf("nodes table should not include latitude after split migration")
	}
	if !hasColumn(t, migrated, "nodes", "channel") {
		t.Fatalf("nodes table should keep channel after split migration")
	}

	for _, table := range []string{
		"node_position_latest",
		"node_position_history",
		"node_telemetry_latest",
		"node_telemetry_history",
		"node_identity_history",
	} {
		if !hasTable(t, migrated, table) {
			t.Fatalf("expected table %s after migration", table)
		}
	}

	assertSingleTableRow(t, migrated, "node_position_latest")
	assertSingleTableRow(t, migrated, "node_position_history")
	assertSingleTableRow(t, migrated, "node_telemetry_latest")
	assertSingleTableRow(t, migrated, "node_telemetry_history")
	assertSingleTableRow(t, migrated, "node_identity_history")
}

func TestOpenMigratesV4ToV12(t *testing.T) {
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
	if version != 12 {
		t.Fatalf("expected schema version 12, got %d", version)
	}
}

func hasTable(t *testing.T, db *sql.DB, tableName string) bool {
	t.Helper()
	var found string
	err := db.QueryRow(`SELECT name FROM sqlite_master WHERE type = 'table' AND name = ?`, tableName).Scan(&found)

	return err == nil && found == tableName
}

func hasColumn(t *testing.T, db *sql.DB, tableName, columnName string) bool {
	t.Helper()
	rows, err := db.Query(`PRAGMA table_info(` + tableName + `);`)
	if err != nil {
		t.Fatalf("read table info for %s: %v", tableName, err)
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
			t.Fatalf("scan table info for %s: %v", tableName, err)
		}
		if name == columnName {
			return true
		}
	}

	return false
}

func assertSingleTableRow(t *testing.T, db *sql.DB, tableName string) {
	t.Helper()
	const want = 1

	var got int
	if err := db.QueryRow(`SELECT COUNT(*) FROM ` + tableName + `;`).Scan(&got); err != nil {
		t.Fatalf("count rows in %s: %v", tableName, err)
	}
	if got != want {
		t.Fatalf("unexpected %s row count: got %d want %d", tableName, got, want)
	}
}
