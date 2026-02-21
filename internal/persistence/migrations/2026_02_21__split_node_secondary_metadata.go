package migrations

import (
	"context"
	"database/sql"
)

func migrateV12SplitNodeSecondaryMetadata(ctx context.Context, tx *sql.Tx) error {
	statements := []string{
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
		 WHERE latitude IS NOT NULL
		   AND longitude IS NOT NULL
		   AND latitude BETWEEN -90 AND 90
		   AND longitude BETWEEN -180 AND 180;`,
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
	}

	return applyStatements(ctx, tx, "v12 split node secondary metadata", statements)
}
