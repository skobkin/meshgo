package persistence

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
	"time"

	"github.com/skobkin/meshgo/internal/domain"
)

// NodeTelemetryRepo persists and queries node telemetry snapshots and history.
type NodeTelemetryRepo struct {
	db *sql.DB
}

func NewNodeTelemetryRepo(db *sql.DB) *NodeTelemetryRepo {
	return &NodeTelemetryRepo{db: db}
}

func (r *NodeTelemetryRepo) Upsert(ctx context.Context, update domain.NodeTelemetryUpdate, historyLimit int) error {
	if r == nil || r.db == nil {
		return fmt.Errorf("node telemetry repo is not initialized")
	}
	nodeID := strings.TrimSpace(update.Telemetry.NodeID)
	if nodeID == "" {
		return nil
	}
	incoming := update.Telemetry
	writtenAt := time.Now()
	if incoming.ObservedAt.IsZero() {
		incoming.ObservedAt = incoming.UpdatedAt
	}
	if incoming.ObservedAt.IsZero() {
		incoming.ObservedAt = writtenAt
	}
	if incoming.UpdatedAt.IsZero() {
		incoming.UpdatedAt = writtenAt
	}

	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin node telemetry upsert tx: %w", err)
	}
	defer func() {
		_ = tx.Rollback()
	}()

	_, err = tx.ExecContext(ctx, `
		INSERT INTO nodes(node_id, last_heard_at, updated_at)
		VALUES (?, ?, ?)
		ON CONFLICT(node_id) DO NOTHING
	`, nodeID, timeToUnixMillis(incoming.ObservedAt), timeToUnixMillis(incoming.UpdatedAt))
	if err != nil {
		return fmt.Errorf("ensure node core row for telemetry: %w", err)
	}

	existing, found, err := fetchNodeTelemetryLatest(ctx, tx, nodeID)
	if err != nil {
		return err
	}
	next := mergeNodeTelemetry(existing, incoming)
	if next.ObservedAt.IsZero() {
		next.ObservedAt = incoming.ObservedAt
	}
	if next.UpdatedAt.IsZero() {
		next.UpdatedAt = incoming.UpdatedAt
	}

	_, err = tx.ExecContext(ctx, `
		INSERT INTO node_telemetry_latest(node_id, channel, battery_level, voltage, uptime_seconds, channel_utilization, air_util_tx, temperature, humidity, pressure, soil_temperature, soil_moisture, gas_resistance, lux, uv_lux, radiation, air_quality_index, power_voltage, power_current, observed_at, written_at, update_type, from_packet)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
		ON CONFLICT(node_id) DO UPDATE SET
			channel = COALESCE(excluded.channel, node_telemetry_latest.channel),
			battery_level = COALESCE(excluded.battery_level, node_telemetry_latest.battery_level),
			voltage = COALESCE(excluded.voltage, node_telemetry_latest.voltage),
			uptime_seconds = COALESCE(excluded.uptime_seconds, node_telemetry_latest.uptime_seconds),
			channel_utilization = COALESCE(excluded.channel_utilization, node_telemetry_latest.channel_utilization),
			air_util_tx = COALESCE(excluded.air_util_tx, node_telemetry_latest.air_util_tx),
			temperature = COALESCE(excluded.temperature, node_telemetry_latest.temperature),
			humidity = COALESCE(excluded.humidity, node_telemetry_latest.humidity),
			pressure = COALESCE(excluded.pressure, node_telemetry_latest.pressure),
			soil_temperature = COALESCE(excluded.soil_temperature, node_telemetry_latest.soil_temperature),
			soil_moisture = COALESCE(excluded.soil_moisture, node_telemetry_latest.soil_moisture),
			gas_resistance = COALESCE(excluded.gas_resistance, node_telemetry_latest.gas_resistance),
			lux = COALESCE(excluded.lux, node_telemetry_latest.lux),
			uv_lux = COALESCE(excluded.uv_lux, node_telemetry_latest.uv_lux),
			radiation = COALESCE(excluded.radiation, node_telemetry_latest.radiation),
			air_quality_index = COALESCE(excluded.air_quality_index, node_telemetry_latest.air_quality_index),
			power_voltage = COALESCE(excluded.power_voltage, node_telemetry_latest.power_voltage),
			power_current = COALESCE(excluded.power_current, node_telemetry_latest.power_current),
			observed_at = excluded.observed_at,
			written_at = excluded.written_at,
			update_type = excluded.update_type,
			from_packet = excluded.from_packet
	`,
		nodeID,
		nullableUint32(next.Channel),
		nullableUint32(next.BatteryLevel),
		nullableFloat64(next.Voltage),
		nullableUint32(next.UptimeSeconds),
		nullableFloat64(next.ChannelUtilization),
		nullableFloat64(next.AirUtilTx),
		nullableFloat64(next.Temperature),
		nullableFloat64(next.Humidity),
		nullableFloat64(next.Pressure),
		nullableFloat64(next.SoilTemperature),
		nullableUint32(next.SoilMoisture),
		nullableFloat64(next.GasResistance),
		nullableFloat64(next.Lux),
		nullableFloat64(next.UVLux),
		nullableFloat64(next.Radiation),
		nullableFloat64(next.AirQualityIndex),
		nullableFloat64(next.PowerVoltage),
		nullableFloat64(next.PowerCurrent),
		timeToUnixMillis(next.ObservedAt),
		timeToUnixMillis(writtenAt),
		string(update.Type),
		boolToInt64(update.FromPacket),
	)
	if err != nil {
		return fmt.Errorf("upsert node telemetry latest: %w", err)
	}

	if hasTelemetryData(next) && (!found || !nodeTelemetryEqual(existing, next)) {
		_, err = tx.ExecContext(ctx, `
				INSERT INTO node_telemetry_history(node_id, channel, battery_level, voltage, uptime_seconds, channel_utilization, air_util_tx, temperature, humidity, pressure, soil_temperature, soil_moisture, gas_resistance, lux, uv_lux, radiation, air_quality_index, power_voltage, power_current, observed_at, written_at, update_type, from_packet)
				VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
			`,
			nodeID,
			nullableUint32(next.Channel),
			nullableUint32(next.BatteryLevel),
			nullableFloat64(next.Voltage),
			nullableUint32(next.UptimeSeconds),
			nullableFloat64(next.ChannelUtilization),
			nullableFloat64(next.AirUtilTx),
			nullableFloat64(next.Temperature),
			nullableFloat64(next.Humidity),
			nullableFloat64(next.Pressure),
			nullableFloat64(next.SoilTemperature),
			nullableUint32(next.SoilMoisture),
			nullableFloat64(next.GasResistance),
			nullableFloat64(next.Lux),
			nullableFloat64(next.UVLux),
			nullableFloat64(next.Radiation),
			nullableFloat64(next.AirQualityIndex),
			nullableFloat64(next.PowerVoltage),
			nullableFloat64(next.PowerCurrent),
			timeToUnixMillis(next.ObservedAt),
			timeToUnixMillis(writtenAt),
			string(update.Type),
			boolToInt64(update.FromPacket),
		)
		if err != nil {
			return fmt.Errorf("insert node telemetry history: %w", err)
		}
		if err := pruneHistoryRows(ctx, tx, "node_telemetry_history", nodeID, historyLimit); err != nil {
			return err
		}
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("commit node telemetry upsert tx: %w", err)
	}

	return nil
}

func (r *NodeTelemetryRepo) ListLatest(ctx context.Context) ([]domain.NodeTelemetry, error) {
	rows, err := r.db.QueryContext(ctx, `
		SELECT node_id, channel, battery_level, voltage, uptime_seconds, channel_utilization, air_util_tx, temperature, humidity, pressure, soil_temperature, soil_moisture, gas_resistance, lux, uv_lux, radiation, air_quality_index, power_voltage, power_current, observed_at, written_at
		FROM node_telemetry_latest
	`)
	if err != nil {
		return nil, fmt.Errorf("list node telemetry latest: %w", err)
	}
	defer func() {
		_ = rows.Close()
	}()

	out := make([]domain.NodeTelemetry, 0)
	for rows.Next() {
		item, scanErr := scanNodeTelemetryLatest(rows)
		if scanErr != nil {
			return nil, scanErr
		}
		out = append(out, item)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate node telemetry latest rows: %w", err)
	}

	return out, nil
}

func (r *NodeTelemetryRepo) GetLatestByNodeID(ctx context.Context, nodeID string) (domain.NodeTelemetry, bool, error) {
	rows, err := r.db.QueryContext(ctx, `
		SELECT node_id, channel, battery_level, voltage, uptime_seconds, channel_utilization, air_util_tx, temperature, humidity, pressure, soil_temperature, soil_moisture, gas_resistance, lux, uv_lux, radiation, air_quality_index, power_voltage, power_current, observed_at, written_at
		FROM node_telemetry_latest
		WHERE node_id = ?
		LIMIT 1
	`, strings.TrimSpace(nodeID))
	if err != nil {
		return domain.NodeTelemetry{}, false, fmt.Errorf("query node telemetry latest by id: %w", err)
	}
	defer func() {
		_ = rows.Close()
	}()
	if !rows.Next() {
		return domain.NodeTelemetry{}, false, nil
	}
	item, scanErr := scanNodeTelemetryLatest(rows)
	if scanErr != nil {
		return domain.NodeTelemetry{}, false, scanErr
	}

	return item, true, nil
}

func (r *NodeTelemetryRepo) ListHistoryByNodeID(ctx context.Context, query domain.NodeHistoryQuery) ([]domain.NodeTelemetryHistoryEntry, error) {
	nodeID := strings.TrimSpace(query.NodeID)
	if nodeID == "" {
		return nil, nil
	}
	order := historyOrderSQL(query.Order)
	where := "WHERE node_id = ?"
	args := []any{nodeID}
	where, args = applyHistoryCursor(where, query, args)
	limit := historyLimitValue(query.Limit)
	rows, err := r.db.QueryContext(ctx, fmt.Sprintf(`
		SELECT id, node_id, channel, battery_level, voltage, uptime_seconds, channel_utilization, air_util_tx, temperature, humidity, pressure, soil_temperature, soil_moisture, gas_resistance, lux, uv_lux, radiation, air_quality_index, power_voltage, power_current, observed_at, written_at, update_type, from_packet
		FROM node_telemetry_history
		%s
		ORDER BY observed_at %s, id %s
		LIMIT ?
	`, where, order, order), append(args, limit)...)
	if err != nil {
		return nil, fmt.Errorf("list node telemetry history: %w", err)
	}
	defer func() {
		_ = rows.Close()
	}()

	out := make([]domain.NodeTelemetryHistoryEntry, 0)
	for rows.Next() {
		item, scanErr := scanNodeTelemetryHistory(rows)
		if scanErr != nil {
			return nil, scanErr
		}
		out = append(out, item)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate node telemetry history rows: %w", err)
	}

	return out, nil
}

func fetchNodeTelemetryLatest(ctx context.Context, tx *sql.Tx, nodeID string) (domain.NodeTelemetry, bool, error) {
	rows, err := tx.QueryContext(ctx, `
		SELECT node_id, channel, battery_level, voltage, uptime_seconds, channel_utilization, air_util_tx, temperature, humidity, pressure, soil_temperature, soil_moisture, gas_resistance, lux, uv_lux, radiation, air_quality_index, power_voltage, power_current, observed_at, written_at
		FROM node_telemetry_latest
		WHERE node_id = ?
		LIMIT 1
	`, nodeID)
	if err != nil {
		return domain.NodeTelemetry{}, false, fmt.Errorf("query existing node telemetry latest: %w", err)
	}
	defer func() {
		_ = rows.Close()
	}()
	if !rows.Next() {
		return domain.NodeTelemetry{}, false, nil
	}
	item, scanErr := scanNodeTelemetryLatest(rows)
	if scanErr != nil {
		return domain.NodeTelemetry{}, false, scanErr
	}

	return item, true, nil
}

func scanNodeTelemetryLatest(scanner interface{ Scan(dest ...any) error }) (domain.NodeTelemetry, error) {
	var (
		item          domain.NodeTelemetry
		channel       sql.NullInt64
		battery       sql.NullInt64
		voltage       sql.NullFloat64
		uptime        sql.NullInt64
		channelUtil   sql.NullFloat64
		airUtilTx     sql.NullFloat64
		temperature   sql.NullFloat64
		humidity      sql.NullFloat64
		pressure      sql.NullFloat64
		soilTemp      sql.NullFloat64
		soilMoisture  sql.NullInt64
		gasResistance sql.NullFloat64
		lux           sql.NullFloat64
		uvLux         sql.NullFloat64
		radiation     sql.NullFloat64
		aqi           sql.NullFloat64
		powerVoltage  sql.NullFloat64
		powerCurrent  sql.NullFloat64
		observedMS    int64
		writtenMS     int64
	)
	if err := scanner.Scan(&item.NodeID, &channel, &battery, &voltage, &uptime, &channelUtil, &airUtilTx, &temperature, &humidity, &pressure, &soilTemp, &soilMoisture, &gasResistance, &lux, &uvLux, &radiation, &aqi, &powerVoltage, &powerCurrent, &observedMS, &writtenMS); err != nil {
		return domain.NodeTelemetry{}, fmt.Errorf("scan node telemetry latest row: %w", err)
	}
	if channel.Valid {
		if v, ok := int64ToUint32(channel.Int64); ok {
			item.Channel = &v
		}
	}
	if battery.Valid {
		if v, ok := int64ToUint32(battery.Int64); ok {
			item.BatteryLevel = &v
		}
	}
	if voltage.Valid {
		v := voltage.Float64
		item.Voltage = &v
	}
	if uptime.Valid {
		if v, ok := int64ToUint32(uptime.Int64); ok {
			item.UptimeSeconds = &v
		}
	}
	if channelUtil.Valid {
		v := channelUtil.Float64
		item.ChannelUtilization = &v
	}
	if airUtilTx.Valid {
		v := airUtilTx.Float64
		item.AirUtilTx = &v
	}
	if temperature.Valid {
		v := temperature.Float64
		item.Temperature = &v
	}
	if humidity.Valid {
		v := humidity.Float64
		item.Humidity = &v
	}
	if pressure.Valid {
		v := pressure.Float64
		item.Pressure = &v
	}
	if soilTemp.Valid {
		v := soilTemp.Float64
		item.SoilTemperature = &v
	}
	if soilMoisture.Valid {
		if v, ok := int64ToUint32(soilMoisture.Int64); ok {
			item.SoilMoisture = &v
		}
	}
	if gasResistance.Valid {
		v := gasResistance.Float64
		item.GasResistance = &v
	}
	if lux.Valid {
		v := lux.Float64
		item.Lux = &v
	}
	if uvLux.Valid {
		v := uvLux.Float64
		item.UVLux = &v
	}
	if radiation.Valid {
		v := radiation.Float64
		item.Radiation = &v
	}
	if aqi.Valid {
		v := aqi.Float64
		item.AirQualityIndex = &v
	}
	if powerVoltage.Valid {
		v := powerVoltage.Float64
		item.PowerVoltage = &v
	}
	if powerCurrent.Valid {
		v := powerCurrent.Float64
		item.PowerCurrent = &v
	}
	item.ObservedAt = unixMillisToTime(observedMS)
	item.UpdatedAt = unixMillisToTime(writtenMS)

	return item, nil
}

func scanNodeTelemetryHistory(scanner interface{ Scan(dest ...any) error }) (domain.NodeTelemetryHistoryEntry, error) {
	var (
		item          domain.NodeTelemetryHistoryEntry
		channel       sql.NullInt64
		battery       sql.NullInt64
		voltage       sql.NullFloat64
		uptime        sql.NullInt64
		channelUtil   sql.NullFloat64
		airUtilTx     sql.NullFloat64
		temperature   sql.NullFloat64
		humidity      sql.NullFloat64
		pressure      sql.NullFloat64
		soilTemp      sql.NullFloat64
		soilMoisture  sql.NullInt64
		gasResistance sql.NullFloat64
		lux           sql.NullFloat64
		uvLux         sql.NullFloat64
		radiation     sql.NullFloat64
		aqi           sql.NullFloat64
		powerVoltage  sql.NullFloat64
		powerCurrent  sql.NullFloat64
		observedMS    int64
		writtenMS     int64
		updateType    string
		fromPacket    int64
	)
	if err := scanner.Scan(&item.RowID, &item.NodeID, &channel, &battery, &voltage, &uptime, &channelUtil, &airUtilTx, &temperature, &humidity, &pressure, &soilTemp, &soilMoisture, &gasResistance, &lux, &uvLux, &radiation, &aqi, &powerVoltage, &powerCurrent, &observedMS, &writtenMS, &updateType, &fromPacket); err != nil {
		return domain.NodeTelemetryHistoryEntry{}, fmt.Errorf("scan node telemetry history row: %w", err)
	}
	if channel.Valid {
		if v, ok := int64ToUint32(channel.Int64); ok {
			item.Channel = &v
		}
	}
	if battery.Valid {
		if v, ok := int64ToUint32(battery.Int64); ok {
			item.BatteryLevel = &v
		}
	}
	if voltage.Valid {
		v := voltage.Float64
		item.Voltage = &v
	}
	if uptime.Valid {
		if v, ok := int64ToUint32(uptime.Int64); ok {
			item.UptimeSeconds = &v
		}
	}
	if channelUtil.Valid {
		v := channelUtil.Float64
		item.ChannelUtilization = &v
	}
	if airUtilTx.Valid {
		v := airUtilTx.Float64
		item.AirUtilTx = &v
	}
	if temperature.Valid {
		v := temperature.Float64
		item.Temperature = &v
	}
	if humidity.Valid {
		v := humidity.Float64
		item.Humidity = &v
	}
	if pressure.Valid {
		v := pressure.Float64
		item.Pressure = &v
	}
	if soilTemp.Valid {
		v := soilTemp.Float64
		item.SoilTemperature = &v
	}
	if soilMoisture.Valid {
		if v, ok := int64ToUint32(soilMoisture.Int64); ok {
			item.SoilMoisture = &v
		}
	}
	if gasResistance.Valid {
		v := gasResistance.Float64
		item.GasResistance = &v
	}
	if lux.Valid {
		v := lux.Float64
		item.Lux = &v
	}
	if uvLux.Valid {
		v := uvLux.Float64
		item.UVLux = &v
	}
	if radiation.Valid {
		v := radiation.Float64
		item.Radiation = &v
	}
	if aqi.Valid {
		v := aqi.Float64
		item.AirQualityIndex = &v
	}
	if powerVoltage.Valid {
		v := powerVoltage.Float64
		item.PowerVoltage = &v
	}
	if powerCurrent.Valid {
		v := powerCurrent.Float64
		item.PowerCurrent = &v
	}
	item.ObservedAt = unixMillisToTime(observedMS)
	item.WrittenAt = unixMillisToTime(writtenMS)
	item.UpdateType = domain.NodeUpdateType(strings.TrimSpace(updateType))
	item.FromPacket = fromPacket != 0

	return item, nil
}

func mergeNodeTelemetry(existing, incoming domain.NodeTelemetry) domain.NodeTelemetry {
	next := existing
	if strings.TrimSpace(next.NodeID) == "" {
		next.NodeID = incoming.NodeID
	}
	if incoming.Channel != nil {
		next.Channel = incoming.Channel
	}
	if incoming.BatteryLevel != nil {
		next.BatteryLevel = incoming.BatteryLevel
	}
	if incoming.Voltage != nil {
		next.Voltage = incoming.Voltage
	}
	if incoming.UptimeSeconds != nil {
		next.UptimeSeconds = incoming.UptimeSeconds
	}
	if incoming.ChannelUtilization != nil {
		next.ChannelUtilization = incoming.ChannelUtilization
	}
	if incoming.AirUtilTx != nil {
		next.AirUtilTx = incoming.AirUtilTx
	}
	if incoming.Temperature != nil {
		next.Temperature = incoming.Temperature
	}
	if incoming.Humidity != nil {
		next.Humidity = incoming.Humidity
	}
	if incoming.Pressure != nil {
		next.Pressure = incoming.Pressure
	}
	if incoming.SoilTemperature != nil {
		next.SoilTemperature = incoming.SoilTemperature
	}
	if incoming.SoilMoisture != nil {
		next.SoilMoisture = incoming.SoilMoisture
	}
	if incoming.GasResistance != nil {
		next.GasResistance = incoming.GasResistance
	}
	if incoming.Lux != nil {
		next.Lux = incoming.Lux
	}
	if incoming.UVLux != nil {
		next.UVLux = incoming.UVLux
	}
	if incoming.Radiation != nil {
		next.Radiation = incoming.Radiation
	}
	if incoming.AirQualityIndex != nil {
		next.AirQualityIndex = incoming.AirQualityIndex
	}
	if incoming.PowerVoltage != nil {
		next.PowerVoltage = incoming.PowerVoltage
	}
	if incoming.PowerCurrent != nil {
		next.PowerCurrent = incoming.PowerCurrent
	}
	if !incoming.ObservedAt.IsZero() {
		next.ObservedAt = incoming.ObservedAt
	}
	if !incoming.UpdatedAt.IsZero() {
		next.UpdatedAt = incoming.UpdatedAt
	}

	return next
}

func hasTelemetryData(value domain.NodeTelemetry) bool {
	return value.BatteryLevel != nil ||
		value.Voltage != nil ||
		value.UptimeSeconds != nil ||
		value.ChannelUtilization != nil ||
		value.AirUtilTx != nil ||
		value.Temperature != nil ||
		value.Humidity != nil ||
		value.Pressure != nil ||
		value.SoilTemperature != nil ||
		value.SoilMoisture != nil ||
		value.GasResistance != nil ||
		value.Lux != nil ||
		value.UVLux != nil ||
		value.Radiation != nil ||
		value.AirQualityIndex != nil ||
		value.PowerVoltage != nil ||
		value.PowerCurrent != nil
}

func nodeTelemetryEqual(left, right domain.NodeTelemetry) bool {
	return nullableUint32Equal(left.Channel, right.Channel) &&
		nullableUint32Equal(left.BatteryLevel, right.BatteryLevel) &&
		nullableFloat64Equal(left.Voltage, right.Voltage) &&
		nullableUint32Equal(left.UptimeSeconds, right.UptimeSeconds) &&
		nullableFloat64Equal(left.ChannelUtilization, right.ChannelUtilization) &&
		nullableFloat64Equal(left.AirUtilTx, right.AirUtilTx) &&
		nullableFloat64Equal(left.Temperature, right.Temperature) &&
		nullableFloat64Equal(left.Humidity, right.Humidity) &&
		nullableFloat64Equal(left.Pressure, right.Pressure) &&
		nullableFloat64Equal(left.SoilTemperature, right.SoilTemperature) &&
		nullableUint32Equal(left.SoilMoisture, right.SoilMoisture) &&
		nullableFloat64Equal(left.GasResistance, right.GasResistance) &&
		nullableFloat64Equal(left.Lux, right.Lux) &&
		nullableFloat64Equal(left.UVLux, right.UVLux) &&
		nullableFloat64Equal(left.Radiation, right.Radiation) &&
		nullableFloat64Equal(left.AirQualityIndex, right.AirQualityIndex) &&
		nullableFloat64Equal(left.PowerVoltage, right.PowerVoltage) &&
		nullableFloat64Equal(left.PowerCurrent, right.PowerCurrent)
}
