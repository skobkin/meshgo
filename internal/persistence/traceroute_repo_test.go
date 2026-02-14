package persistence

import (
	"context"
	"database/sql"
	"path/filepath"
	"testing"
	"time"

	"github.com/skobkin/meshgo/internal/domain"
	"github.com/skobkin/meshgo/internal/traceroute"
)

func TestTracerouteRepoUpsert_PersistsAndUpdatesRecord(t *testing.T) {
	ctx := context.Background()
	dbPath := filepath.Join(t.TempDir(), "app.db")

	db, err := Open(ctx, dbPath)
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	defer func() { _ = db.Close() }()

	repo := NewTracerouteRepo(db)
	startedAt := time.Now().UTC().Add(-3 * time.Second).Truncate(time.Millisecond)
	updatedAt := startedAt.Add(2 * time.Second)

	initial := domain.TracerouteRecord{
		RequestID:    "42",
		TargetNodeID: "!00000042",
		StartedAt:    startedAt,
		UpdatedAt:    updatedAt,
		Status:       traceroute.StatusProgress,
		ForwardRoute: []string{"!00000042", "!00000010"},
		ForwardSNR:   []int32{35},
	}
	if err := repo.Upsert(ctx, initial); err != nil {
		t.Fatalf("upsert initial traceroute: %v", err)
	}

	completedAt := updatedAt.Add(time.Second)
	completed := initial
	completed.UpdatedAt = completedAt
	completed.CompletedAt = completedAt
	completed.Status = traceroute.StatusCompleted
	completed.ReturnRoute = []string{"!00000010", "!00000042"}
	completed.ReturnSNR = []int32{29}
	completed.DurationMS = completedAt.Sub(startedAt).Milliseconds()
	if err := repo.Upsert(ctx, completed); err != nil {
		t.Fatalf("upsert completed traceroute: %v", err)
	}

	var (
		status         string
		targetNodeID   string
		startedMS      int64
		updatedMS      int64
		completedMS    sql.NullInt64
		forwardRouteJS sql.NullString
		returnRouteJS  sql.NullString
	)
	if err := db.QueryRowContext(ctx, `
		SELECT status, target_node_id, started_at, updated_at, completed_at, forward_route_json, return_route_json
		FROM traceroutes
		WHERE request_id = ?
	`, completed.RequestID).Scan(&status, &targetNodeID, &startedMS, &updatedMS, &completedMS, &forwardRouteJS, &returnRouteJS); err != nil {
		t.Fatalf("query traceroute row: %v", err)
	}

	if status != string(traceroute.StatusCompleted) {
		t.Fatalf("unexpected status: %q", status)
	}
	if targetNodeID != completed.TargetNodeID {
		t.Fatalf("unexpected target node id: %q", targetNodeID)
	}
	if startedMS != completed.StartedAt.UnixMilli() {
		t.Fatalf("unexpected started_at: got %d want %d", startedMS, completed.StartedAt.UnixMilli())
	}
	if updatedMS != completed.UpdatedAt.UnixMilli() {
		t.Fatalf("unexpected updated_at: got %d want %d", updatedMS, completed.UpdatedAt.UnixMilli())
	}
	if !completedMS.Valid || completedMS.Int64 != completed.CompletedAt.UnixMilli() {
		t.Fatalf("unexpected completed_at: %+v", completedMS)
	}
	if !forwardRouteJS.Valid || forwardRouteJS.String == "" {
		t.Fatalf("expected forward route json to be set")
	}
	if !returnRouteJS.Valid || returnRouteJS.String == "" {
		t.Fatalf("expected return route json to be set")
	}
}
