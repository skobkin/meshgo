package app

import (
	"context"
	"errors"
	"testing"
	"time"

	generated "github.com/skobkin/meshgo/internal/radio/meshtasticpb"
)

func TestNodeSettingsServiceLoadOwner_RetriesDeadlineExceeded(t *testing.T) {
	t.Parallel()

	sendCalls := uint32(0)
	sender := stubAdminSender{
		send: func(_ uint32, _ uint32, wantResponse bool, payload *generated.AdminMessage) (string, error) {
			sendCalls++
			if !wantResponse {
				t.Fatalf("expected response for load owner")
			}
			if !payload.GetGetOwnerRequest() {
				t.Fatalf("expected get owner request")
			}

			return stringFromUint32(sendCalls), nil
		},
	}
	service, _ := newTestNodeSettingsService(t, sender, false)

	ctx, cancel := context.WithTimeout(context.Background(), time.Millisecond)
	defer cancel()

	_, err := service.loadOwner(ctx, mustHexNodeTarget())
	if err == nil {
		t.Fatalf("expected deadline error")
	}
	if !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("expected deadline exceeded, got %v", err)
	}
	if sendCalls != uint32(nodeSettingsReadRetry+1) {
		t.Fatalf("unexpected send calls: got %d want %d", sendCalls, nodeSettingsReadRetry+1)
	}
}
