package migrations

import (
	"context"
	"database/sql"
)

func migrateV4AddNodeEnvironmentTelemetry(ctx context.Context, tx *sql.Tx) error {
	statements := []string{
		`ALTER TABLE nodes ADD COLUMN temperature REAL NULL;`,
		`ALTER TABLE nodes ADD COLUMN humidity REAL NULL;`,
		`ALTER TABLE nodes ADD COLUMN pressure REAL NULL;`,
		`ALTER TABLE nodes ADD COLUMN air_quality_index REAL NULL;`,
		`ALTER TABLE nodes ADD COLUMN power_voltage REAL NULL;`,
		`ALTER TABLE nodes ADD COLUMN power_current REAL NULL;`,
	}

	return applyStatements(ctx, tx, "v4 add node environment telemetry", statements)
}
