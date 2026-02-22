package migrations

import (
	"context"
	"database/sql"
)

func migrateV13AddExtendedEnvironmentTelemetry(ctx context.Context, tx *sql.Tx) error {
	statements := []string{
		`ALTER TABLE node_telemetry_latest ADD COLUMN gas_resistance REAL NULL;`,
		`ALTER TABLE node_telemetry_latest ADD COLUMN lux REAL NULL;`,
		`ALTER TABLE node_telemetry_latest ADD COLUMN soil_temperature REAL NULL;`,
		`ALTER TABLE node_telemetry_latest ADD COLUMN soil_moisture INTEGER NULL;`,
		`ALTER TABLE node_telemetry_latest ADD COLUMN uv_lux REAL NULL;`,
		`ALTER TABLE node_telemetry_latest ADD COLUMN radiation REAL NULL;`,
		`ALTER TABLE node_telemetry_history ADD COLUMN gas_resistance REAL NULL;`,
		`ALTER TABLE node_telemetry_history ADD COLUMN lux REAL NULL;`,
		`ALTER TABLE node_telemetry_history ADD COLUMN soil_temperature REAL NULL;`,
		`ALTER TABLE node_telemetry_history ADD COLUMN soil_moisture INTEGER NULL;`,
		`ALTER TABLE node_telemetry_history ADD COLUMN uv_lux REAL NULL;`,
		`ALTER TABLE node_telemetry_history ADD COLUMN radiation REAL NULL;`,
	}

	return applyStatements(ctx, tx, "v13 add extended environment telemetry", statements)
}
