package transport

import (
	"bytes"
	"errors"
	"io"
	"math"
	"testing"
)

func TestReadFrameResyncsToHeader(t *testing.T) {
	want := []byte{0x01, 0x02, 0x03}
	raw := bytes.NewBuffer([]byte{
		0x00, 0x11, 0x22, // noise before the frame
		frameHeader[0], frameHeader[1],
		0x00, 0x03,
		0x01, 0x02, 0x03,
	})

	got, err := readFrame(ioReadFullFunc(raw))
	if err != nil {
		t.Fatalf("read frame: %v", err)
	}
	if !bytes.Equal(got, want) {
		t.Fatalf("payload mismatch: got %x want %x", got, want)
	}
}

func TestReadFrameRejectsZeroLength(t *testing.T) {
	raw := bytes.NewBuffer([]byte{
		frameHeader[0], frameHeader[1],
		0x00, 0x00,
	})

	_, err := readFrame(ioReadFullFunc(raw))
	if err == nil {
		t.Fatalf("expected error for zero-length frame, got nil")
	}
}

func TestEncodeFramePayloadTooLarge(t *testing.T) {
	payload := make([]byte, math.MaxUint16+1)
	_, err := encodeFrame(payload)
	if err == nil {
		t.Fatalf("expected payload size error, got nil")
	}
}

func TestEncodeFrameAndReadFrameRoundTrip(t *testing.T) {
	payload := []byte("hello")
	frame, err := encodeFrame(payload)
	if err != nil {
		t.Fatalf("encode frame: %v", err)
	}

	got, err := readFrame(ioReadFullFunc(bytes.NewReader(frame)))
	if err != nil {
		t.Fatalf("read frame: %v", err)
	}
	if !bytes.Equal(got, payload) {
		t.Fatalf("payload mismatch: got %q want %q", string(got), string(payload))
	}
}

func TestReadFramePayloadEOF(t *testing.T) {
	raw := bytes.NewBuffer([]byte{
		frameHeader[0], frameHeader[1],
		0x00, 0x04,
		0x01, 0x02,
	})

	_, err := readFrame(ioReadFullFunc(raw))
	if err == nil {
		t.Fatalf("expected payload read error, got nil")
	}
	if errors.Is(err, io.EOF) {
		t.Fatalf("expected wrapped error, got raw io.EOF")
	}
}
