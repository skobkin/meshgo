package ui

import (
	"bytes"
	"testing"

	"github.com/skobkin/meshgo/internal/app"
	"github.com/skobkin/meshgo/internal/resources"
)

func TestParseNodeChannelPSK(t *testing.T) {
	t.Parallel()

	valid16 := bytes.Repeat([]byte{0x11}, 16)
	valid16B64 := encodeNodeSettingsKeyBase64(valid16)
	got, err := parseNodeChannelPSK(valid16B64)
	if err != nil {
		t.Fatalf("parse valid PSK: %v", err)
	}
	if !bytes.Equal(got, valid16) {
		t.Fatalf("unexpected decoded PSK")
	}

	if _, err := parseNodeChannelPSK("not-base64"); err == nil {
		t.Fatalf("expected parse error for invalid base64")
	}
	if _, err := parseNodeChannelPSK(encodeNodeSettingsKeyBase64(bytes.Repeat([]byte{0x22}, 12))); err == nil {
		t.Fatalf("expected parse error for invalid PSK size")
	}
}

func TestNodeChannelEncryptionIcon(t *testing.T) {
	t.Parallel()

	if got := nodeChannelEncryptionIcon(app.NodeChannelSettings{PSK: bytes.Repeat([]byte{0x33}, 16)}); got != resources.UIIconLockGreen {
		t.Fatalf("expected lock green, got %s", got)
	}
	if got := nodeChannelEncryptionIcon(app.NodeChannelSettings{PSK: []byte{1}, PositionPrecision: 0, UplinkEnabled: false}); got != resources.UIIconLockYellow {
		t.Fatalf("expected lock yellow, got %s", got)
	}
	if got := nodeChannelEncryptionIcon(app.NodeChannelSettings{PSK: []byte{1}, PositionPrecision: 32, UplinkEnabled: false}); got != resources.UIIconLockRed {
		t.Fatalf("expected lock red, got %s", got)
	}
	if got := nodeChannelEncryptionIcon(app.NodeChannelSettings{PSK: []byte{1}, PositionPrecision: 32, UplinkEnabled: true}); got != resources.UIIconLockRedWarning {
		t.Fatalf("expected lock red warning, got %s", got)
	}
}

func TestParseNodeChannelPositionPrecision(t *testing.T) {
	t.Parallel()

	if got, err := parseNodeChannelPositionPrecision("Disabled"); err != nil || got != 0 {
		t.Fatalf("expected disabled precision, got %d err=%v", got, err)
	}
	if got, err := parseNodeChannelPositionPrecision("Precise"); err != nil || got != 32 {
		t.Fatalf("expected precise precision, got %d err=%v", got, err)
	}
	if got, err := parseNodeChannelPositionPrecision("Precise location (32)"); err != nil || got != 32 {
		t.Fatalf("expected legacy precise precision, got %d err=%v", got, err)
	}
	label13 := nodeChannelPositionPrecisionLabel(13)
	if got, err := parseNodeChannelPositionPrecision(label13); err != nil || got != 13 {
		t.Fatalf("expected precision 13 from label %q, got %d err=%v", label13, got, err)
	}
	if _, err := parseNodeChannelPositionPrecision("invalid"); err == nil {
		t.Fatalf("expected parse error for invalid precision label")
	}
}

func TestNodeChannelDisplayTitle(t *testing.T) {
	t.Parallel()

	if got := nodeChannelDisplayTitle(app.NodeChannelSettings{Name: " Ops "}, 1, "LongFast"); got != "Ops" {
		t.Fatalf("expected explicit channel name, got %q", got)
	}
	if got := nodeChannelDisplayTitle(app.NodeChannelSettings{}, 1, "LongFast"); got != "LongFast" {
		t.Fatalf("expected preset fallback title, got %q", got)
	}
	if got := nodeChannelDisplayTitle(app.NodeChannelSettings{}, 1, ""); got != "Channel 2" {
		t.Fatalf("expected index fallback for secondary, got %q", got)
	}
}
