package storage

import (
	"context"
	"database/sql"

	"meshgo/domain"
	_ "modernc.org/sqlite"
)

// NodeStore persists node information to SQLite.
type NodeStore struct {
	db *sql.DB
}

// OpenNodeStore opens the SQLite database at the provided path.
func OpenNodeStore(path string) (*NodeStore, error) {
	db, err := sql.Open("sqlite", path)
	if err != nil {
		return nil, err
	}
	return &NodeStore{db: db}, nil
}

// Init ensures the schema exists.
func (s *NodeStore) Init(ctx context.Context) error {
	schema := `CREATE TABLE IF NOT EXISTS nodes (
        id TEXT PRIMARY KEY,
        short_name TEXT,
        long_name TEXT,
        favorite BOOLEAN,
        ignored BOOLEAN,
        unencrypted BOOLEAN,
        enc_default_key BOOLEAN,
        enc_custom_key BOOLEAN,
        rssi INTEGER,
        snr REAL,
        signal_quality INTEGER,
        battery_level INTEGER,
        is_charging BOOLEAN,
        last_heard INTEGER
    );`
	_, err := s.db.ExecContext(ctx, schema)
	return err
}

// UpsertNode inserts or updates a node record.
func (s *NodeStore) UpsertNode(ctx context.Context, n *domain.Node) error {
	var batt interface{}
	if n.BatteryLevel != nil {
		batt = *n.BatteryLevel
	}
	var chg interface{}
	if n.IsCharging != nil {
		chg = *n.IsCharging
	}
	_, err := s.db.ExecContext(ctx, `INSERT INTO nodes (
        id, short_name, long_name, favorite, ignored, unencrypted, enc_default_key, enc_custom_key,
        rssi, snr, signal_quality, battery_level, is_charging, last_heard
    ) VALUES (?,?,?,?,?,?,?,?,?,?,?,?,?,?)
    ON CONFLICT(id) DO UPDATE SET
        short_name=excluded.short_name,
        long_name=excluded.long_name,
        favorite=excluded.favorite,
        ignored=excluded.ignored,
        unencrypted=excluded.unencrypted,
        enc_default_key=excluded.enc_default_key,
        enc_custom_key=excluded.enc_custom_key,
        rssi=excluded.rssi,
        snr=excluded.snr,
        signal_quality=excluded.signal_quality,
        battery_level=excluded.battery_level,
        is_charging=excluded.is_charging,
        last_heard=excluded.last_heard`,
		n.ID, n.ShortName, n.LongName, n.Favorite, n.Ignored, n.Unencrypted, n.EncDefaultKey, n.EncCustomKey,
		n.RSSI, n.SNR, int(n.Signal), batt, chg, n.LastHeard)
	return err
}

// SetFavorite marks a node as favorite or not.
func (s *NodeStore) SetFavorite(ctx context.Context, id string, fav bool) error {
	_, err := s.db.ExecContext(ctx, `UPDATE nodes SET favorite=? WHERE id=?`, fav, id)
	return err
}

// SetIgnored marks a node as ignored or not.
func (s *NodeStore) SetIgnored(ctx context.Context, id string, ignored bool) error {
	_, err := s.db.ExecContext(ctx, `UPDATE nodes SET ignored=? WHERE id=?`, ignored, id)
	return err
}

// RemoveNode deletes the node with the given ID.
func (s *NodeStore) RemoveNode(ctx context.Context, id string) error {
	_, err := s.db.ExecContext(ctx, `DELETE FROM nodes WHERE id=?`, id)
	return err
}

// ListNodes returns all nodes in the store.
func (s *NodeStore) ListNodes(ctx context.Context) ([]*domain.Node, error) {
	rows, err := s.db.QueryContext(ctx, `SELECT id, short_name, long_name, favorite, ignored, unencrypted, enc_default_key, enc_custom_key, rssi, snr, signal_quality, battery_level, is_charging, last_heard FROM nodes`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var nodes []*domain.Node
	for rows.Next() {
		var n domain.Node
		var signal int
		var batt sql.NullInt64
		var chg sql.NullBool
		if err := rows.Scan(&n.ID, &n.ShortName, &n.LongName, &n.Favorite, &n.Ignored, &n.Unencrypted, &n.EncDefaultKey, &n.EncCustomKey, &n.RSSI, &n.SNR, &signal, &batt, &chg, &n.LastHeard); err != nil {
			return nil, err
		}
		n.Signal = domain.SignalQuality(signal)
		if batt.Valid {
			v := int(batt.Int64)
			n.BatteryLevel = &v
		}
		if chg.Valid {
			v := chg.Bool
			n.IsCharging = &v
		}
		nodes = append(nodes, &n)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return nodes, nil
}

// Close closes the underlying database.
func (s *NodeStore) Close() error { return s.db.Close() }
