package ui

import (
	"encoding/base64"
	"strings"
)

// encodeNodeSettingsKeyBase64 converts raw key bytes to a textual form suitable
// for node settings forms and clipboard copy actions.
func encodeNodeSettingsKeyBase64(raw []byte) string {
	if len(raw) == 0 {
		return ""
	}

	return base64.StdEncoding.EncodeToString(raw)
}

// decodeNodeSettingsKeyBase64 accepts both padded and raw base64 variants from
// text inputs, so pasted keys from different tools are handled consistently.
func decodeNodeSettingsKeyBase64(raw string) ([]byte, error) {
	decoded, err := base64.StdEncoding.DecodeString(raw)
	if err == nil {
		return decoded, nil
	}

	decoded, rawErr := base64.RawStdEncoding.DecodeString(raw)
	if rawErr == nil {
		return decoded, nil
	}

	return nil, err
}

// formatNodeSettingsKeysPerLine renders each key as one base64 line.
// This representation is shared by Security admin keys and is reusable for
// future text-based key editors such as Channels encryption keys.
func formatNodeSettingsKeysPerLine(keys [][]byte) string {
	if len(keys) == 0 {
		return ""
	}

	parts := make([]string, 0, len(keys))
	for _, key := range keys {
		encoded := encodeNodeSettingsKeyBase64(key)
		if encoded == "" {
			continue
		}
		parts = append(parts, encoded)
	}

	return strings.Join(parts, "\n")
}

// cloneNodeSettingsKeyBytes returns a detached copy of key bytes to avoid
// accidental aliasing between form state and baseline/device-loaded state.
func cloneNodeSettingsKeyBytes(value []byte) []byte {
	if len(value) == 0 {
		return nil
	}

	out := make([]byte, len(value))
	copy(out, value)

	return out
}

// cloneNodeSettingsKeyBytesList returns detached copies for each key in a list.
func cloneNodeSettingsKeyBytesList(values [][]byte) [][]byte {
	if len(values) == 0 {
		return nil
	}

	out := make([][]byte, 0, len(values))
	for _, value := range values {
		out = append(out, cloneNodeSettingsKeyBytes(value))
	}

	return out
}
