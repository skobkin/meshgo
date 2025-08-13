package transport

import (
	"context"
	"net"
	"testing"
	"time"
)

func TestTCPEndpoint(t *testing.T) {
	t.Parallel()
	tt := NewTCP("localhost:1234")
	if got, want := tt.Endpoint(), "localhost:1234"; got != want {
		t.Fatalf("endpoint = %q, want %q", got, want)
	}
}

func TestTCPReadWrite(t *testing.T) {
	t.Parallel()
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("listen: %v", err)
	}
	defer ln.Close()
	addr := ln.Addr().String()

	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()

	done := make(chan struct{})
	go func() {
		conn, err := ln.Accept()
		if err != nil {
			return
		}
		defer conn.Close()
		buf := make([]byte, 4)
		if _, err := conn.Read(buf); err == nil {
			conn.Write([]byte("pong"))
		}
		close(done)
	}()

	tt := NewTCP(addr)
	if err := tt.Connect(ctx); err != nil {
		t.Fatalf("connect: %v", err)
	}
	if !tt.IsConnected() {
		t.Fatalf("transport should report connected")
	}
	if err := tt.WritePacket(ctx, []byte("ping")); err != nil {
		t.Fatalf("write: %v", err)
	}
	if b, err := tt.ReadPacket(ctx); err != nil || string(b) != "pong" {
		t.Fatalf("read = %q, err %v", string(b), err)
	}
	if err := tt.Close(); err != nil {
		t.Fatalf("close: %v", err)
	}
	if tt.IsConnected() {
		t.Fatalf("transport should report disconnected after close")
	}
	<-done
}
