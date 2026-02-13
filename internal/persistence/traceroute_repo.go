package persistence

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"

	"github.com/skobkin/meshgo/internal/domain"
)

// TracerouteRepo implements domain.TracerouteRepository using SQLite.
type TracerouteRepo struct {
	db *sql.DB
}

func NewTracerouteRepo(db *sql.DB) *TracerouteRepo {
	return &TracerouteRepo{db: db}
}

func (r *TracerouteRepo) Upsert(ctx context.Context, rec domain.TracerouteRecord) error {
	forwardRouteJSON, err := marshalJSONNullable(rec.ForwardRoute)
	if err != nil {
		return fmt.Errorf("marshal forward route: %w", err)
	}
	forwardSNRJSON, err := marshalJSONNullable(rec.ForwardSNR)
	if err != nil {
		return fmt.Errorf("marshal forward snr: %w", err)
	}
	returnRouteJSON, err := marshalJSONNullable(rec.ReturnRoute)
	if err != nil {
		return fmt.Errorf("marshal return route: %w", err)
	}
	returnSNRJSON, err := marshalJSONNullable(rec.ReturnSNR)
	if err != nil {
		return fmt.Errorf("marshal return snr: %w", err)
	}

	_, err = r.db.ExecContext(ctx, `
		INSERT INTO traceroutes(
			request_id, target_node_id, started_at, updated_at, completed_at, status,
			forward_route_json, forward_snr_json, return_route_json, return_snr_json, error_text, duration_ms
		)
		VALUES(?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
		ON CONFLICT(request_id) DO UPDATE SET
			target_node_id = excluded.target_node_id,
			started_at = excluded.started_at,
			updated_at = excluded.updated_at,
			completed_at = excluded.completed_at,
			status = excluded.status,
			forward_route_json = excluded.forward_route_json,
			forward_snr_json = excluded.forward_snr_json,
			return_route_json = excluded.return_route_json,
			return_snr_json = excluded.return_snr_json,
			error_text = excluded.error_text,
			duration_ms = excluded.duration_ms
	`,
		rec.RequestID,
		rec.TargetNodeID,
		timeToUnixMillis(rec.StartedAt),
		timeToUnixMillis(rec.UpdatedAt),
		timeToUnixMillis(rec.CompletedAt),
		string(rec.Status),
		forwardRouteJSON,
		forwardSNRJSON,
		returnRouteJSON,
		returnSNRJSON,
		nullableString(rec.ErrorText),
		rec.DurationMS,
	)
	if err != nil {
		return fmt.Errorf("upsert traceroute: %w", err)
	}

	return nil
}

func marshalJSONNullable(v any) (any, error) {
	raw, err := json.Marshal(v)
	if err != nil {
		return nil, err
	}
	if string(raw) == "null" || string(raw) == "[]" {
		return nil, nil
	}

	return string(raw), nil
}
