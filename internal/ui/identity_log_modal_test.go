package ui

import (
	"encoding/base64"
	"testing"
	"time"

	"github.com/skobkin/meshgo/internal/domain"
)

func TestIdentityLogRow(t *testing.T) {
	observed := time.Date(2026, 2, 1, 12, 0, 0, 0, time.UTC)
	publicKey := []byte{1, 2, 3, 4}
	row := identityLogRow(domain.NodeIdentityHistoryEntry{
		ObservedAt: observed,
		UpdateType: domain.NodeUpdateTypeNodeInfoPacket,
		FromPacket: true,
		LongName:   "Long",
		ShortName:  "SH",
		PublicKey:  publicKey,
	})
	if got := row[0]; got == "unknown" {
		t.Fatalf("expected observed time value")
	}
	if got := row[1]; got != string(domain.NodeUpdateTypeNodeInfoPacket) {
		t.Fatalf("unexpected update type value: %q", got)
	}
	if got := row[2]; got != "yes" {
		t.Fatalf("unexpected from packet value: %q", got)
	}
	if got := row[3]; got != "Long" {
		t.Fatalf("unexpected long name value: %q", got)
	}
	if got := row[4]; got != "SH" {
		t.Fatalf("unexpected short name value: %q", got)
	}
	if got := row[5]; got != base64.StdEncoding.EncodeToString(publicKey) {
		t.Fatalf("unexpected public key value: %q", got)
	}
}

func TestIdentityLogRowUnknownDefaults(t *testing.T) {
	row := identityLogRow(domain.NodeIdentityHistoryEntry{})
	if got := row[3]; got != "unknown" {
		t.Fatalf("unexpected long name fallback: %q", got)
	}
	if got := row[4]; got != "unknown" {
		t.Fatalf("unexpected short name fallback: %q", got)
	}
	if got := row[5]; got != "unknown" {
		t.Fatalf("unexpected public key fallback: %q", got)
	}
}
