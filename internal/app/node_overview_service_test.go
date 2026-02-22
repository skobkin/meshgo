package app

import (
	"context"
	"errors"
	"testing"

	"github.com/skobkin/meshgo/internal/domain"
	"github.com/skobkin/meshgo/internal/radio"
	"github.com/skobkin/meshgo/internal/radio/busmsg"
	generated "github.com/skobkin/meshgo/internal/radio/meshtasticpb"
)

type nodeOverviewRadioSpy struct {
	nodeInfoTo        uint32
	nodeInfoChannel   uint32
	nodeInfoRequester *generated.User
	nodeInfoErr       error

	telemetryTo      uint32
	telemetryChannel uint32
	telemetryKind    radio.TelemetryRequestKind
	telemetryErr     error
}

func (s *nodeOverviewRadioSpy) SendNodeInfoRequest(to uint32, channel uint32, requester *generated.User) (string, error) {
	s.nodeInfoTo = to
	s.nodeInfoChannel = channel
	s.nodeInfoRequester = requester
	if s.nodeInfoErr != nil {
		return "", s.nodeInfoErr
	}

	return "11", nil
}

func (s *nodeOverviewRadioSpy) SendTelemetryRequest(to uint32, channel uint32, kind radio.TelemetryRequestKind) (string, error) {
	s.telemetryTo = to
	s.telemetryChannel = channel
	s.telemetryKind = kind
	if s.telemetryErr != nil {
		return "", s.telemetryErr
	}

	return "22", nil
}

type telemetryRepoSpy struct {
	items     []domain.NodeTelemetryHistoryEntry
	err       error
	lastQuery domain.NodeHistoryQuery
}

func (s *telemetryRepoSpy) Upsert(context.Context, domain.NodeTelemetryUpdate, int) error {
	return nil
}

func (s *telemetryRepoSpy) ListLatest(context.Context) ([]domain.NodeTelemetry, error) {
	return nil, nil
}

func (s *telemetryRepoSpy) GetLatestByNodeID(context.Context, string) (domain.NodeTelemetry, bool, error) {
	return domain.NodeTelemetry{}, false, nil
}

func (s *telemetryRepoSpy) ListHistoryByNodeID(_ context.Context, query domain.NodeHistoryQuery) ([]domain.NodeTelemetryHistoryEntry, error) {
	s.lastQuery = query
	if s.err != nil {
		return nil, s.err
	}

	return s.items, nil
}

func TestNodeOverviewServiceRequestUserInfo(t *testing.T) {
	store := domain.NewNodeStore()
	channel := uint32(5)
	store.Upsert(domain.Node{NodeID: "!0000002a", Channel: &channel})
	spy := &nodeOverviewRadioSpy{}
	service := NewNodeOverviewService(
		spy,
		store,
		&telemetryRepoSpy{},
		func() (busmsg.ConnectionStatus, bool) {
			return busmsg.ConnectionStatus{State: busmsg.ConnectionStateConnected}, true
		},
		nil,
	)

	err := service.RequestUserInfo(context.Background(), "!0000002a", LocalNodeSnapshot{
		ID: "!00000001",
		Node: domain.Node{
			LongName:  "Local",
			ShortName: "LOC",
		},
	})
	if err != nil {
		t.Fatalf("request user info: %v", err)
	}
	if spy.nodeInfoTo != 0x2a {
		t.Fatalf("unexpected target node num: %d", spy.nodeInfoTo)
	}
	if spy.nodeInfoChannel != 5 {
		t.Fatalf("unexpected target channel: %d", spy.nodeInfoChannel)
	}
	if spy.nodeInfoRequester == nil {
		t.Fatalf("expected requester payload")
	}
	if spy.nodeInfoRequester.GetId() != "!00000001" {
		t.Fatalf("unexpected requester id: %q", spy.nodeInfoRequester.GetId())
	}
}

func TestNodeOverviewServiceRequestTelemetry(t *testing.T) {
	spy := &nodeOverviewRadioSpy{}
	service := NewNodeOverviewService(
		spy,
		domain.NewNodeStore(),
		&telemetryRepoSpy{},
		func() (busmsg.ConnectionStatus, bool) {
			return busmsg.ConnectionStatus{State: busmsg.ConnectionStateConnected}, true
		},
		nil,
	)

	err := service.RequestTelemetry(context.Background(), "!0000002a", radio.TelemetryRequestPower)
	if err != nil {
		t.Fatalf("request telemetry: %v", err)
	}
	if spy.telemetryTo != 0x2a {
		t.Fatalf("unexpected telemetry target node num: %d", spy.telemetryTo)
	}
	if spy.telemetryChannel != 0 {
		t.Fatalf("unexpected telemetry channel fallback: %d", spy.telemetryChannel)
	}
	if spy.telemetryKind != radio.TelemetryRequestPower {
		t.Fatalf("unexpected telemetry kind: %q", spy.telemetryKind)
	}
}

func TestNodeOverviewServiceRequestFailsWhenDisconnected(t *testing.T) {
	service := NewNodeOverviewService(
		&nodeOverviewRadioSpy{},
		domain.NewNodeStore(),
		&telemetryRepoSpy{},
		func() (busmsg.ConnectionStatus, bool) {
			return busmsg.ConnectionStatus{State: busmsg.ConnectionStateDisconnected}, true
		},
		nil,
	)

	if err := service.RequestUserInfo(context.Background(), "!0000002a", LocalNodeSnapshot{ID: "!00000001"}); err == nil {
		t.Fatalf("expected connection error for user info request")
	}
	if err := service.RequestTelemetry(context.Background(), "!0000002a", radio.TelemetryRequestDevice); err == nil {
		t.Fatalf("expected connection error for telemetry request")
	}
}

func TestNodeOverviewServiceRequestTelemetryPropagatesRadioError(t *testing.T) {
	wantErr := errors.New("write failed")
	spy := &nodeOverviewRadioSpy{telemetryErr: wantErr}
	service := NewNodeOverviewService(
		spy,
		domain.NewNodeStore(),
		&telemetryRepoSpy{},
		func() (busmsg.ConnectionStatus, bool) {
			return busmsg.ConnectionStatus{State: busmsg.ConnectionStateConnected}, true
		},
		nil,
	)

	err := service.RequestTelemetry(context.Background(), "!0000002a", radio.TelemetryRequestDevice)
	if !errors.Is(err, wantErr) {
		t.Fatalf("expected telemetry send error, got %v", err)
	}
}

func TestNodeOverviewServiceListTelemetryHistory(t *testing.T) {
	repo := &telemetryRepoSpy{
		items: []domain.NodeTelemetryHistoryEntry{
			{RowID: 10, NodeID: "!0000002a"},
		},
	}
	service := NewNodeOverviewService(
		&nodeOverviewRadioSpy{},
		domain.NewNodeStore(),
		repo,
		func() (busmsg.ConnectionStatus, bool) { return busmsg.ConnectionStatus{}, false },
		nil,
	)

	items, err := service.ListTelemetryHistory(context.Background(), "!0000002a", 0)
	if err != nil {
		t.Fatalf("list telemetry history: %v", err)
	}
	if len(items) != 1 {
		t.Fatalf("expected one telemetry entry, got %d", len(items))
	}
	if repo.lastQuery.NodeID != "!0000002a" {
		t.Fatalf("unexpected query node id: %q", repo.lastQuery.NodeID)
	}
	if repo.lastQuery.Limit != defaultNodeOverviewTelemetryHistoryLimit {
		t.Fatalf("unexpected default query limit: %d", repo.lastQuery.Limit)
	}
	if repo.lastQuery.Order != domain.SortDescending {
		t.Fatalf("unexpected query order: %q", repo.lastQuery.Order)
	}
}
