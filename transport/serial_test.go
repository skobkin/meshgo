package transport

import (
	"context"
	"testing"
)

func TestSerialEndpoint(t *testing.T) {
	t.Parallel()
	s := NewSerial("/dev/testport")
	if got, want := s.Endpoint(), "/dev/testport"; got != want {
		t.Fatalf("endpoint = %q, want %q", got, want)
	}
}

func TestSerialConnectInvalidPort(t *testing.T) {
	t.Parallel()
	s := NewSerial("not-a-real-port")
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	if err := s.Connect(ctx); err == nil {
		t.Fatalf("expected error connecting to invalid port")
	}
	if s.IsConnected() {
		t.Fatalf("transport should not report connected after failed connect")
	}
}
