package transport

import (
	"encoding/binary"
	"fmt"
	"io"
	"math"
)

var frameHeader = [2]byte{0x94, 0xC3}

type readFullFunc func(buf []byte) error

func encodeFrame(payload []byte) ([]byte, error) {
	if len(payload) > math.MaxUint16 {
		return nil, fmt.Errorf("payload too large: %d", len(payload))
	}

	frame := make([]byte, 4+len(payload))
	frame[0] = frameHeader[0]
	frame[1] = frameHeader[1]
	// #nosec G115 -- length is bounded by math.MaxUint16 above.
	payloadLen := uint16(len(payload))
	binary.BigEndian.PutUint16(frame[2:4], payloadLen)
	copy(frame[4:], payload)

	return frame, nil
}

func readFrame(readFull readFullFunc) ([]byte, error) {
	if err := resyncToHeader(readFull); err != nil {
		return nil, err
	}

	var lenBuf [2]byte
	if err := readFull(lenBuf[:]); err != nil {
		return nil, fmt.Errorf("read frame length: %w", err)
	}
	ln := int(binary.BigEndian.Uint16(lenBuf[:]))
	if ln <= 0 {
		return nil, fmt.Errorf("invalid frame length: %d", ln)
	}

	payload := make([]byte, ln)
	if err := readFull(payload); err != nil {
		return nil, fmt.Errorf("read frame payload: %w", err)
	}

	return payload, nil
}

func resyncToHeader(readFull readFullFunc) error {
	buf := make([]byte, 1)
	for {
		if err := readFull(buf); err != nil {
			return fmt.Errorf("read frame header byte 1: %w", err)
		}
		if buf[0] != frameHeader[0] {
			continue
		}
		if err := readFull(buf); err != nil {
			return fmt.Errorf("read frame header byte 2: %w", err)
		}
		if buf[0] == frameHeader[1] {
			return nil
		}
	}
}

func ioReadFullFunc(r io.Reader) readFullFunc {
	return func(buf []byte) error {
		_, err := io.ReadFull(r, buf)

		return err
	}
}
