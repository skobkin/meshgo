package app

import (
	"context"
	"io"
	"log/slog"
	"testing"
	"time"

	"github.com/skobkin/meshgo/internal/bus"
	"github.com/skobkin/meshgo/internal/domain"
	"github.com/skobkin/meshgo/internal/radio/busmsg"
	generated "github.com/skobkin/meshgo/internal/radio/meshtasticpb"
)

func TestNodeMetadataProjection_PublishesNodeUpdateFromAdminMetadata(t *testing.T) {
	messageBus := bus.New(slog.New(slog.NewTextHandler(io.Discard, nil)))
	proj := NewNodeMetadataProjection()
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	proj.Start(ctx, messageBus)

	sub := messageBus.Subscribe(bus.TopicNodeInfo)
	defer messageBus.Unsubscribe(sub, bus.TopicNodeInfo)

	messageBus.Publish(bus.TopicAdminMessage, busmsg.AdminMessageEvent{
		From: 0x1234abcd,
		Message: &generated.AdminMessage{
			PayloadVariant: &generated.AdminMessage_GetDeviceMetadataResponse{
				GetDeviceMetadataResponse: &generated.DeviceMetadata{
					FirmwareVersion: "2.5.1.99999",
					HwModel:         generated.HardwareModel_TBEAM,
				},
			},
		},
	})

	select {
	case raw := <-sub:
		update, ok := raw.(domain.NodeUpdate)
		if !ok {
			t.Fatalf("unexpected payload type: %T", raw)
		}
		if update.Type != domain.NodeUpdateTypeMetadata {
			t.Fatalf("unexpected update type: %q", update.Type)
		}
		if update.Node.NodeID != "!1234abcd" {
			t.Fatalf("unexpected node id: %q", update.Node.NodeID)
		}
		if update.Node.FirmwareVersion != "2.5.1.99999" {
			t.Fatalf("unexpected firmware version: %q", update.Node.FirmwareVersion)
		}
		if update.Node.BoardModel != generated.HardwareModel_TBEAM.String() {
			t.Fatalf("unexpected board model: %q", update.Node.BoardModel)
		}
	case <-time.After(2 * time.Second):
		t.Fatalf("timed out waiting for node metadata update")
	}
}
