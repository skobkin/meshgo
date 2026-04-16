package app

import (
	"context"
	"testing"

	"github.com/skobkin/meshgo/internal/bus"
	generated "github.com/skobkin/meshgo/internal/radio/meshtasticpb"
)

func TestNodeSettingsServiceMaintenanceActions_SendExpectedAdminPayloads(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name  string
		run   func(context.Context, *NodeSettingsService) error
		check func(*testing.T, *generated.AdminMessage)
	}{
		{
			name: "Reboot",
			run: func(ctx context.Context, service *NodeSettingsService) error {
				return service.RebootNode(ctx, mustLocalNodeTarget())
			},
			check: func(t *testing.T, payload *generated.AdminMessage) {
				if payload.GetRebootSeconds() != 1 {
					t.Fatalf("unexpected reboot payload: %+v", payload)
				}
			},
		},
		{
			name: "Shutdown",
			run: func(ctx context.Context, service *NodeSettingsService) error {
				return service.ShutdownNode(ctx, mustLocalNodeTarget())
			},
			check: func(t *testing.T, payload *generated.AdminMessage) {
				if payload.GetShutdownSeconds() != 1 {
					t.Fatalf("unexpected shutdown payload: %+v", payload)
				}
			},
		},
		{
			name: "FactoryReset",
			run: func(ctx context.Context, service *NodeSettingsService) error {
				return service.FactoryResetNode(ctx, mustLocalNodeTarget())
			},
			check: func(t *testing.T, payload *generated.AdminMessage) {
				if payload.GetFactoryResetConfig() != 1 {
					t.Fatalf("unexpected factory reset payload: %+v", payload)
				}
			},
		},
		{
			name: "ResetNodeDB",
			run: func(ctx context.Context, service *NodeSettingsService) error {
				return service.ResetNodeDB(ctx, mustLocalNodeTarget(), true)
			},
			check: func(t *testing.T, payload *generated.AdminMessage) {
				if !payload.GetNodedbReset() {
					t.Fatalf("unexpected node db reset payload: %+v", payload)
				}
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			var messageBus bus.MessageBus
			sender := stubAdminSender{
				send: func(_ uint32, _ uint32, wantResponse bool, payload *generated.AdminMessage) (string, error) {
					if wantResponse {
						t.Fatalf("expected maintenance actions to send without wantResponse")
					}
					tc.check(t, payload)
					publishSentStatus(messageBus, 1)

					return "1", nil
				},
			}
			service, busRef := newTestNodeSettingsService(t, sender, true)
			messageBus = busRef

			ctx, cancel := contextWithTimeout(t)
			defer cancel()
			if err := tc.run(ctx, service); err != nil {
				t.Fatalf("maintenance action failed: %v", err)
			}
		})
	}
}
