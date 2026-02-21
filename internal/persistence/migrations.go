package persistence

import (
	"context"
	"database/sql"
	"fmt"
	"log/slog"
)

const schemaVersion = 12

type schemaMigration struct {
	version int
	name    string
	apply   func(context.Context, *sql.DB) error
}

var schemaMigrations = []schemaMigration{
	{version: 11, name: "legacy_to_v11", apply: migrateLegacyToV11},
	{version: 12, name: "split_node_secondary_metadata", apply: migrateV12SplitNodeSecondaryMetadata},
}

func migrate(ctx context.Context, db *sql.DB) error {
	var version int
	if err := db.QueryRowContext(ctx, `PRAGMA user_version;`).Scan(&version); err != nil {
		return fmt.Errorf("read schema version: %w", err)
	}
	slog.Info("db schema version detected", "current", version, "target", schemaVersion)

	if version >= schemaVersion {
		slog.Info("db schema is up to date", "version", version)

		return nil
	}

	for _, migration := range schemaMigrations {
		if version >= migration.version {
			continue
		}
		slog.Info("applying db migration", "from", version, "to", migration.version, "name", migration.name)
		if err := migration.apply(ctx, db); err != nil {
			return fmt.Errorf("apply migration %s (%d): %w", migration.name, migration.version, err)
		}
		version = migration.version
	}

	slog.Info("db migration completed", "version", version)

	return nil
}

func migrateLegacyToV11(ctx context.Context, db *sql.DB) error {
	var version int
	if err := db.QueryRowContext(ctx, `PRAGMA user_version;`).Scan(&version); err != nil {
		return fmt.Errorf("read schema version before v11 migration: %w", err)
	}
	if version >= 11 {
		return nil
	}

	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin v11 migration tx: %w", err)
	}
	defer func() {
		_ = tx.Rollback()
	}()

	if version < 1 {
		stmts := []string{
			`CREATE TABLE IF NOT EXISTS nodes (
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
			`CREATE INDEX IF NOT EXISTS nodes_last_heard_at_idx ON nodes(last_heard_at DESC);`,
			`CREATE TABLE IF NOT EXISTS chats (
				chat_key TEXT PRIMARY KEY,
				type INTEGER NOT NULL,
				title TEXT NOT NULL,
				last_sent_by_me_at INTEGER NULL,
				updated_at INTEGER NOT NULL
			);`,
			`CREATE INDEX IF NOT EXISTS chats_last_sent_by_me_idx ON chats(last_sent_by_me_at DESC);`,
			`CREATE TABLE IF NOT EXISTS messages (
				local_id INTEGER PRIMARY KEY AUTOINCREMENT,
				chat_key TEXT NOT NULL,
				device_message_id TEXT NULL,
				reply_to_device_message_id TEXT NULL,
				emoji INTEGER NOT NULL DEFAULT 0,
				direction INTEGER NOT NULL,
				body TEXT NOT NULL,
				status INTEGER NOT NULL,
				at INTEGER NOT NULL,
				meta_json TEXT NULL
			);`,
			`CREATE INDEX IF NOT EXISTS messages_chat_at_idx ON messages(chat_key, at ASC);`,
			`CREATE UNIQUE INDEX IF NOT EXISTS messages_chat_device_unique_idx ON messages(chat_key, device_message_id) WHERE device_message_id IS NOT NULL;`,
			`CREATE TABLE IF NOT EXISTS traceroutes (
				request_id TEXT PRIMARY KEY,
				target_node_id TEXT NOT NULL,
				started_at INTEGER NOT NULL,
				updated_at INTEGER NOT NULL,
				completed_at INTEGER NULL,
				status TEXT NOT NULL,
				forward_route_json TEXT NULL,
				forward_snr_json TEXT NULL,
				return_route_json TEXT NULL,
				return_snr_json TEXT NULL,
				error_text TEXT NULL,
				duration_ms INTEGER NULL
			);`,
			`CREATE INDEX IF NOT EXISTS traceroutes_started_at_idx ON traceroutes(started_at DESC);`,
			`CREATE INDEX IF NOT EXISTS traceroutes_target_started_idx ON traceroutes(target_node_id, started_at DESC);`,
			`PRAGMA user_version = 11;`,
		}
		for _, stmt := range stmts {
			if _, err := tx.ExecContext(ctx, stmt); err != nil {
				return fmt.Errorf("apply v11 bootstrap statement: %w", err)
			}
		}
	} else {
		if version < 2 {
			stmts := []string{
				`ALTER TABLE nodes ADD COLUMN battery_level INTEGER NULL;`,
				`ALTER TABLE nodes ADD COLUMN voltage REAL NULL;`,
				`ALTER TABLE nodes ADD COLUMN board_model TEXT NULL;`,
				`ALTER TABLE nodes ADD COLUMN device_role TEXT NULL;`,
				`PRAGMA user_version = 2;`,
			}
			for _, stmt := range stmts {
				if _, err := tx.ExecContext(ctx, stmt); err != nil {
					return fmt.Errorf("apply v2 migration statement: %w", err)
				}
			}
			version = 2
		}
		if version < 3 {
			stmts := []string{
				`ALTER TABLE nodes ADD COLUMN is_unmessageable INTEGER NULL;`,
				`PRAGMA user_version = 3;`,
			}
			for _, stmt := range stmts {
				if _, err := tx.ExecContext(ctx, stmt); err != nil {
					return fmt.Errorf("apply v3 migration statement: %w", err)
				}
			}
			version = 3
		}
		if version < 4 {
			stmts := []string{
				`ALTER TABLE nodes ADD COLUMN temperature REAL NULL;`,
				`ALTER TABLE nodes ADD COLUMN humidity REAL NULL;`,
				`ALTER TABLE nodes ADD COLUMN pressure REAL NULL;`,
				`ALTER TABLE nodes ADD COLUMN air_quality_index REAL NULL;`,
				`ALTER TABLE nodes ADD COLUMN power_voltage REAL NULL;`,
				`ALTER TABLE nodes ADD COLUMN power_current REAL NULL;`,
				`PRAGMA user_version = 4;`,
			}
			for _, stmt := range stmts {
				if _, err := tx.ExecContext(ctx, stmt); err != nil {
					return fmt.Errorf("apply v4 migration statement: %w", err)
				}
			}
			version = 4
		}
		if version < 5 {
			stmts := []string{
				`ALTER TABLE nodes ADD COLUMN latitude REAL NULL;`,
				`ALTER TABLE nodes ADD COLUMN longitude REAL NULL;`,
				`PRAGMA user_version = 5;`,
			}
			for _, stmt := range stmts {
				if _, err := tx.ExecContext(ctx, stmt); err != nil {
					return fmt.Errorf("apply v5 migration statement: %w", err)
				}
			}
			version = 5
		}
		if version < 6 {
			stmts := []string{
				`ALTER TABLE nodes ADD COLUMN channel INTEGER NULL;`,
				`CREATE TABLE IF NOT EXISTS traceroutes (
					request_id TEXT PRIMARY KEY,
					target_node_id TEXT NOT NULL,
					started_at INTEGER NOT NULL,
					updated_at INTEGER NOT NULL,
					completed_at INTEGER NULL,
					status TEXT NOT NULL,
					forward_route_json TEXT NULL,
					forward_snr_json TEXT NULL,
					return_route_json TEXT NULL,
					return_snr_json TEXT NULL,
					error_text TEXT NULL,
					duration_ms INTEGER NULL
				);`,
				`CREATE INDEX IF NOT EXISTS traceroutes_started_at_idx ON traceroutes(started_at DESC);`,
				`CREATE INDEX IF NOT EXISTS traceroutes_target_started_idx ON traceroutes(target_node_id, started_at DESC);`,
				`PRAGMA user_version = 6;`,
			}
			for _, stmt := range stmts {
				if _, err := tx.ExecContext(ctx, stmt); err != nil {
					return fmt.Errorf("apply v6 migration statement: %w", err)
				}
			}
			version = 6
		}
		if version < 7 {
			stmts := []string{
				`ALTER TABLE nodes ADD COLUMN altitude INTEGER NULL;`,
				`PRAGMA user_version = 7;`,
			}
			for _, stmt := range stmts {
				if _, err := tx.ExecContext(ctx, stmt); err != nil {
					return fmt.Errorf("apply v7 migration statement: %w", err)
				}
			}
			version = 7
		}
		if version < 8 {
			stmts := []string{
				`ALTER TABLE messages ADD COLUMN reply_to_device_message_id TEXT NULL;`,
				`ALTER TABLE messages ADD COLUMN emoji INTEGER NOT NULL DEFAULT 0;`,
				`PRAGMA user_version = 8;`,
			}
			for _, stmt := range stmts {
				if _, err := tx.ExecContext(ctx, stmt); err != nil {
					return fmt.Errorf("apply v8 migration statement: %w", err)
				}
			}
			version = 8
		}
		if version < 9 {
			stmts := []string{
				`ALTER TABLE nodes ADD COLUMN precision_bits INTEGER NULL;`,
				`PRAGMA user_version = 9;`,
			}
			for _, stmt := range stmts {
				if _, err := tx.ExecContext(ctx, stmt); err != nil {
					return fmt.Errorf("apply v9 migration statement: %w", err)
				}
			}
			version = 9
		}
		if version < 10 {
			stmts := []string{
				`ALTER TABLE nodes ADD COLUMN uptime_seconds INTEGER NULL;`,
				`ALTER TABLE nodes ADD COLUMN channel_utilization REAL NULL;`,
				`ALTER TABLE nodes ADD COLUMN air_util_tx REAL NULL;`,
				`ALTER TABLE nodes ADD COLUMN firmware_version TEXT NULL;`,
				`ALTER TABLE nodes ADD COLUMN position_updated_at INTEGER NULL;`,
				`PRAGMA user_version = 10;`,
			}
			for _, stmt := range stmts {
				if _, err := tx.ExecContext(ctx, stmt); err != nil {
					return fmt.Errorf("apply v10 migration statement: %w", err)
				}
			}
			version = 10
		}
		if version < 11 {
			stmts := []string{
				`ALTER TABLE nodes ADD COLUMN public_key BLOB NULL;`,
				`PRAGMA user_version = 11;`,
			}
			for _, stmt := range stmts {
				if _, err := tx.ExecContext(ctx, stmt); err != nil {
					return fmt.Errorf("apply v11 migration statement: %w", err)
				}
			}
		}
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("commit v11 migration: %w", err)
	}

	return nil
}

func migrateV12SplitNodeSecondaryMetadata(ctx context.Context, db *sql.DB) error {
	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin v12 migration tx: %w", err)
	}
	defer func() {
		_ = tx.Rollback()
	}()

	stmts := []string{
		`ALTER TABLE nodes RENAME TO nodes_legacy;`,
		`DROP INDEX IF EXISTS nodes_last_heard_at_idx;`,
		`CREATE TABLE nodes (
			node_id TEXT PRIMARY KEY,
			long_name TEXT,
			short_name TEXT,
			public_key BLOB NULL,
			channel INTEGER NULL,
			board_model TEXT NULL,
			firmware_version TEXT NULL,
			device_role TEXT NULL,
			is_unmessageable INTEGER NULL,
			last_heard_at INTEGER NOT NULL,
			rssi INTEGER NULL,
			snr REAL NULL,
			updated_at INTEGER NOT NULL
		);`,
		`INSERT INTO nodes(node_id, long_name, short_name, public_key, channel, board_model, firmware_version, device_role, is_unmessageable, last_heard_at, rssi, snr, updated_at)
		 SELECT node_id, long_name, short_name, public_key, channel, board_model, firmware_version, device_role, is_unmessageable, last_heard_at, rssi, snr, updated_at
		 FROM nodes_legacy;`,
		`CREATE INDEX nodes_last_heard_at_idx ON nodes(last_heard_at DESC);`,
		`CREATE TABLE node_position_latest (
			node_id TEXT PRIMARY KEY,
			channel INTEGER NULL,
			latitude REAL NULL,
			longitude REAL NULL,
			altitude INTEGER NULL,
			precision_bits INTEGER NULL,
			position_updated_at INTEGER NULL,
			observed_at INTEGER NOT NULL,
			written_at INTEGER NOT NULL,
			update_type TEXT NOT NULL,
			from_packet INTEGER NOT NULL,
			FOREIGN KEY(node_id) REFERENCES nodes(node_id) ON DELETE CASCADE
		);`,
		`CREATE TABLE node_position_history (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			node_id TEXT NOT NULL,
			channel INTEGER NULL,
			latitude REAL NULL,
			longitude REAL NULL,
			altitude INTEGER NULL,
			precision_bits INTEGER NULL,
			position_updated_at INTEGER NULL,
			observed_at INTEGER NOT NULL,
			written_at INTEGER NOT NULL,
			update_type TEXT NOT NULL,
			from_packet INTEGER NOT NULL,
			FOREIGN KEY(node_id) REFERENCES nodes(node_id) ON DELETE CASCADE
		);`,
		`CREATE INDEX node_position_history_node_observed_idx ON node_position_history(node_id, observed_at DESC, id DESC);`,
		`CREATE TABLE node_telemetry_latest (
			node_id TEXT PRIMARY KEY,
			channel INTEGER NULL,
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
			observed_at INTEGER NOT NULL,
			written_at INTEGER NOT NULL,
			update_type TEXT NOT NULL,
			from_packet INTEGER NOT NULL,
			FOREIGN KEY(node_id) REFERENCES nodes(node_id) ON DELETE CASCADE
		);`,
		`CREATE TABLE node_telemetry_history (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			node_id TEXT NOT NULL,
			channel INTEGER NULL,
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
			observed_at INTEGER NOT NULL,
			written_at INTEGER NOT NULL,
			update_type TEXT NOT NULL,
			from_packet INTEGER NOT NULL,
			FOREIGN KEY(node_id) REFERENCES nodes(node_id) ON DELETE CASCADE
		);`,
		`CREATE INDEX node_telemetry_history_node_observed_idx ON node_telemetry_history(node_id, observed_at DESC, id DESC);`,
		`CREATE TABLE node_identity_history (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			node_id TEXT NOT NULL,
			long_name TEXT NULL,
			short_name TEXT NULL,
			public_key BLOB NULL,
			observed_at INTEGER NOT NULL,
			written_at INTEGER NOT NULL,
			update_type TEXT NOT NULL,
			from_packet INTEGER NOT NULL,
			FOREIGN KEY(node_id) REFERENCES nodes(node_id) ON DELETE CASCADE
		);`,
		`CREATE INDEX node_identity_history_node_observed_idx ON node_identity_history(node_id, observed_at DESC, id DESC);`,
		`INSERT INTO node_position_latest(node_id, channel, latitude, longitude, altitude, precision_bits, position_updated_at, observed_at, written_at, update_type, from_packet)
		 SELECT node_id,
				channel,
				latitude,
				longitude,
				altitude,
				precision_bits,
				position_updated_at,
				COALESCE(position_updated_at, last_heard_at, updated_at),
				updated_at,
				'migration_baseline',
				0
		 FROM nodes_legacy
		 WHERE latitude IS NOT NULL OR longitude IS NOT NULL OR altitude IS NOT NULL OR precision_bits IS NOT NULL OR position_updated_at IS NOT NULL;`,
		`INSERT INTO node_position_history(node_id, channel, latitude, longitude, altitude, precision_bits, position_updated_at, observed_at, written_at, update_type, from_packet)
		 SELECT node_id, channel, latitude, longitude, altitude, precision_bits, position_updated_at, observed_at, written_at, update_type, from_packet
		 FROM node_position_latest;`,
		`INSERT INTO node_telemetry_latest(node_id, channel, battery_level, voltage, uptime_seconds, channel_utilization, air_util_tx, temperature, humidity, pressure, air_quality_index, power_voltage, power_current, observed_at, written_at, update_type, from_packet)
		 SELECT node_id,
				channel,
				battery_level,
				voltage,
				uptime_seconds,
				channel_utilization,
				air_util_tx,
				temperature,
				humidity,
				pressure,
				air_quality_index,
				power_voltage,
				power_current,
				COALESCE(last_heard_at, updated_at),
				updated_at,
				'migration_baseline',
				0
		 FROM nodes_legacy
		 WHERE battery_level IS NOT NULL OR voltage IS NOT NULL OR uptime_seconds IS NOT NULL OR channel_utilization IS NOT NULL OR air_util_tx IS NOT NULL OR temperature IS NOT NULL OR humidity IS NOT NULL OR pressure IS NOT NULL OR air_quality_index IS NOT NULL OR power_voltage IS NOT NULL OR power_current IS NOT NULL;`,
		`INSERT INTO node_telemetry_history(node_id, channel, battery_level, voltage, uptime_seconds, channel_utilization, air_util_tx, temperature, humidity, pressure, air_quality_index, power_voltage, power_current, observed_at, written_at, update_type, from_packet)
		 SELECT node_id, channel, battery_level, voltage, uptime_seconds, channel_utilization, air_util_tx, temperature, humidity, pressure, air_quality_index, power_voltage, power_current, observed_at, written_at, update_type, from_packet
		 FROM node_telemetry_latest;`,
		`INSERT INTO node_identity_history(node_id, long_name, short_name, public_key, observed_at, written_at, update_type, from_packet)
		 SELECT node_id,
				NULLIF(long_name, ''),
				NULLIF(short_name, ''),
				public_key,
				COALESCE(last_heard_at, updated_at),
				updated_at,
				'migration_baseline',
				0
		 FROM nodes_legacy
		 WHERE long_name <> '' OR short_name <> '' OR (public_key IS NOT NULL AND length(public_key) > 0);`,
		`DROP TABLE nodes_legacy;`,
		`PRAGMA user_version = 12;`,
	}
	for _, stmt := range stmts {
		if _, err := tx.ExecContext(ctx, stmt); err != nil {
			return fmt.Errorf("apply v12 migration statement: %w", err)
		}
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("commit v12 migration: %w", err)
	}

	return nil
}
