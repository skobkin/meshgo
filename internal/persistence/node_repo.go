package persistence

import (
	"context"
	"database/sql"
	"fmt"
	"math"

	"github.com/skobkin/meshgo/internal/domain"
)

// NodeRepo implements domain.NodeRepository using SQLite.
type NodeRepo struct {
	db *sql.DB
}

func NewNodeRepo(db *sql.DB) *NodeRepo {
	return &NodeRepo{db: db}
}

func (r *NodeRepo) Upsert(ctx context.Context, n domain.Node) error {
	var (
		batteryLevel    any
		voltage         any
		temperature     any
		humidity        any
		pressure        any
		airQualityIndex any
		powerVoltage    any
		powerCurrent    any
		isUnmessageable any
	)
	if n.BatteryLevel != nil {
		batteryLevel = int64(*n.BatteryLevel)
	}
	if n.Voltage != nil {
		voltage = *n.Voltage
	}
	if n.Temperature != nil {
		temperature = *n.Temperature
	}
	if n.Humidity != nil {
		humidity = *n.Humidity
	}
	if n.Pressure != nil {
		pressure = *n.Pressure
	}
	if n.AirQualityIndex != nil {
		airQualityIndex = *n.AirQualityIndex
	}
	if n.PowerVoltage != nil {
		powerVoltage = *n.PowerVoltage
	}
	if n.PowerCurrent != nil {
		powerCurrent = *n.PowerCurrent
	}
	if n.IsUnmessageable != nil {
		if *n.IsUnmessageable {
			isUnmessageable = int64(1)
		} else {
			isUnmessageable = int64(0)
		}
	}
	_, err := r.db.ExecContext(ctx, `
		INSERT INTO nodes(node_id, long_name, short_name, battery_level, voltage, temperature, humidity, pressure, air_quality_index, power_voltage, power_current, board_model, device_role, is_unmessageable, last_heard_at, rssi, snr, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
		ON CONFLICT(node_id) DO UPDATE SET
			long_name = CASE
				WHEN excluded.long_name IS NOT NULL AND excluded.long_name <> '' THEN excluded.long_name
				ELSE nodes.long_name
			END,
			short_name = CASE
				WHEN excluded.short_name IS NOT NULL AND excluded.short_name <> '' THEN excluded.short_name
				ELSE nodes.short_name
			END,
			battery_level = COALESCE(excluded.battery_level, nodes.battery_level),
			voltage = COALESCE(excluded.voltage, nodes.voltage),
			temperature = COALESCE(excluded.temperature, nodes.temperature),
			humidity = COALESCE(excluded.humidity, nodes.humidity),
			pressure = COALESCE(excluded.pressure, nodes.pressure),
			air_quality_index = COALESCE(excluded.air_quality_index, nodes.air_quality_index),
			power_voltage = COALESCE(excluded.power_voltage, nodes.power_voltage),
			power_current = COALESCE(excluded.power_current, nodes.power_current),
			board_model = CASE
				WHEN excluded.board_model IS NOT NULL AND excluded.board_model <> '' THEN excluded.board_model
				ELSE nodes.board_model
			END,
			device_role = CASE
				WHEN excluded.device_role IS NOT NULL AND excluded.device_role <> '' THEN excluded.device_role
				ELSE nodes.device_role
			END,
			is_unmessageable = COALESCE(excluded.is_unmessageable, nodes.is_unmessageable),
			last_heard_at = CASE
				WHEN excluded.last_heard_at > nodes.last_heard_at THEN excluded.last_heard_at
				ELSE nodes.last_heard_at
			END,
			rssi = COALESCE(excluded.rssi, nodes.rssi),
			snr = COALESCE(excluded.snr, nodes.snr),
			updated_at = CASE
				WHEN excluded.updated_at > nodes.updated_at THEN excluded.updated_at
				ELSE nodes.updated_at
			END
	`, n.NodeID, n.LongName, n.ShortName, batteryLevel, voltage, temperature, humidity, pressure, airQualityIndex, powerVoltage, powerCurrent, nullableString(n.BoardModel), nullableString(n.Role), isUnmessageable, timeToUnixMillis(n.LastHeardAt), n.RSSI, n.SNR, timeToUnixMillis(n.UpdatedAt))
	if err != nil {
		return fmt.Errorf("upsert node: %w", err)
	}

	return nil
}

func (r *NodeRepo) ListSortedByLastHeard(ctx context.Context) ([]domain.Node, error) {
	rows, err := r.db.QueryContext(ctx, `
		SELECT node_id, long_name, short_name, battery_level, voltage, temperature, humidity, pressure, air_quality_index, power_voltage, power_current, board_model, device_role, is_unmessageable, last_heard_at, rssi, snr, updated_at
		FROM nodes
		ORDER BY last_heard_at DESC
	`)
	if err != nil {
		return nil, fmt.Errorf("list nodes: %w", err)
	}
	defer func() {
		_ = rows.Close()
	}()

	var out []domain.Node
	for rows.Next() {
		var (
			n             domain.Node
			heardMs       int64
			updMs         int64
			battery       sql.NullInt64
			voltage       sql.NullFloat64
			temperature   sql.NullFloat64
			humidity      sql.NullFloat64
			pressure      sql.NullFloat64
			aqi           sql.NullFloat64
			powerVoltage  sql.NullFloat64
			powerCurrent  sql.NullFloat64
			board         sql.NullString
			role          sql.NullString
			unmessageable sql.NullInt64
			rssi          sql.NullInt64
			snr           sql.NullFloat64
		)
		if err := rows.Scan(&n.NodeID, &n.LongName, &n.ShortName, &battery, &voltage, &temperature, &humidity, &pressure, &aqi, &powerVoltage, &powerCurrent, &board, &role, &unmessageable, &heardMs, &rssi, &snr, &updMs); err != nil {
			return nil, fmt.Errorf("scan node: %w", err)
		}
		n.LastHeardAt = unixMillisToTime(heardMs)
		n.UpdatedAt = unixMillisToTime(updMs)
		if battery.Valid && battery.Int64 >= 0 && battery.Int64 <= math.MaxUint32 {
			// #nosec G115 -- guarded by explicit int64 bounds check.
			v := uint32(battery.Int64)
			n.BatteryLevel = &v
		}
		if voltage.Valid {
			v := voltage.Float64
			n.Voltage = &v
		}
		if temperature.Valid {
			v := temperature.Float64
			n.Temperature = &v
		}
		if humidity.Valid {
			v := humidity.Float64
			n.Humidity = &v
		}
		if pressure.Valid {
			v := pressure.Float64
			n.Pressure = &v
		}
		if aqi.Valid {
			v := aqi.Float64
			n.AirQualityIndex = &v
		}
		if powerVoltage.Valid {
			v := powerVoltage.Float64
			n.PowerVoltage = &v
		}
		if powerCurrent.Valid {
			v := powerCurrent.Float64
			n.PowerCurrent = &v
		}
		if board.Valid {
			n.BoardModel = board.String
		}
		if role.Valid {
			n.Role = role.String
		}
		if unmessageable.Valid {
			v := unmessageable.Int64 != 0
			n.IsUnmessageable = &v
		}
		if rssi.Valid {
			v := int(rssi.Int64)
			n.RSSI = &v
		}
		if snr.Valid {
			v := snr.Float64
			n.SNR = &v
		}
		out = append(out, n)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate nodes: %w", err)
	}

	return out, nil
}
