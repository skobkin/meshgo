package migrations

import (
	"context"
	"database/sql"
	"fmt"
	"log/slog"
)

const targetSchemaVersion = 13

type migrationStep struct {
	version int
	name    string
	apply   func(context.Context, *sql.Tx) error
}

var schemaMigrations = []migrationStep{
	{version: 1, name: "bootstrap_core_schema", apply: migrateV1BootstrapCoreSchema},
	{version: 2, name: "add_node_power_and_device_metadata", apply: migrateV2AddNodePowerAndDeviceMetadata},
	{version: 3, name: "add_node_unmessageable_flag", apply: migrateV3AddNodeUnmessageableFlag},
	{version: 4, name: "add_node_environment_telemetry", apply: migrateV4AddNodeEnvironmentTelemetry},
	{version: 5, name: "add_node_coordinates", apply: migrateV5AddNodeCoordinates},
	{version: 6, name: "add_channel_and_traceroutes", apply: migrateV6AddNodeChannelAndTraceroutes},
	{version: 7, name: "add_node_altitude", apply: migrateV7AddNodeAltitude},
	{version: 8, name: "add_message_replies_and_reactions", apply: migrateV8AddMessageRepliesAndReactions},
	{version: 9, name: "add_node_precision_bits", apply: migrateV9AddNodePrecisionBits},
	{version: 10, name: "add_node_runtime_and_firmware_fields", apply: migrateV10AddNodeRuntimeAndFirmwareFields},
	{version: 11, name: "add_node_public_key", apply: migrateV11AddNodePublicKey},
	{version: 12, name: "split_node_secondary_metadata", apply: migrateV12SplitNodeSecondaryMetadata},
	{version: 13, name: "add_extended_environment_telemetry", apply: migrateV13AddExtendedEnvironmentTelemetry},
}

func Apply(ctx context.Context, db *sql.DB) error {
	version, err := readSchemaVersion(ctx, db)
	if err != nil {
		return err
	}

	slog.Info("db schema version detected", "current", version, "target", targetSchemaVersion)

	if version >= targetSchemaVersion {
		slog.Info("db schema is up to date", "version", version)

		return nil
	}

	for _, migration := range schemaMigrations {
		if version >= migration.version {
			continue
		}

		slog.Info(
			"applying db migration",
			"from", version,
			"to", migration.version,
			"name", migration.name,
		)

		if err := applyMigration(ctx, db, migration); err != nil {
			return fmt.Errorf("apply migration %s (%d): %w", migration.name, migration.version, err)
		}

		version = migration.version
	}

	slog.Info("db migration completed", "version", version)

	return nil
}

func applyMigration(ctx context.Context, db *sql.DB, migration migrationStep) error {
	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin migration tx: %w", err)
	}
	defer func() {
		_ = tx.Rollback()
	}()

	if err := migration.apply(ctx, tx); err != nil {
		return err
	}

	if err := setSchemaVersion(ctx, tx, migration.version); err != nil {
		return err
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("commit migration tx: %w", err)
	}

	return nil
}

func readSchemaVersion(ctx context.Context, db *sql.DB) (int, error) {
	var version int
	if err := db.QueryRowContext(ctx, `PRAGMA user_version;`).Scan(&version); err != nil {
		return 0, fmt.Errorf("read schema version: %w", err)
	}

	return version, nil
}

func setSchemaVersion(ctx context.Context, tx *sql.Tx, version int) error {
	if _, err := tx.ExecContext(ctx, fmt.Sprintf(`PRAGMA user_version = %d;`, version)); err != nil {
		return fmt.Errorf("set schema version %d: %w", version, err)
	}

	return nil
}

func applyStatements(ctx context.Context, tx *sql.Tx, migrationName string, statements []string) error {
	for _, stmt := range statements {
		if _, err := tx.ExecContext(ctx, stmt); err != nil {
			return fmt.Errorf("apply %s statement: %w", migrationName, err)
		}
	}

	return nil
}
