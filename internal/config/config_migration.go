package config

import (
	"encoding/json"
)

// migrateDeprecatedConnector checks for deprecated "connector" field in raw JSON
// and migrates it to "transport" if transport field was not present in the JSON.
//
// Deprecated: This migration function will be removed in a future version.
// It exists only to support migrating from the old "connector" config field to
// the new "transport" field. Planned removal: approximately 2 months from Feb 2026.
func migrateDeprecatedConnector(raw []byte, cfg AppConfig) AppConfig {
	// Use a map to check which fields are actually present in the JSON
	var rawMap map[string]interface{}
	if err := json.Unmarshal(raw, &rawMap); err != nil {
		return cfg
	}

	connectionMap, ok := rawMap["connection"].(map[string]interface{})
	if !ok {
		return cfg
	}

	// Check if "transport" field is present in the JSON
	_, hasTransport := connectionMap["transport"]

	// Check for deprecated "connector" field
	var deprecatedConnector TransportType
	if connectorVal, ok := connectionMap["connector"].(string); ok {
		deprecatedConnector = TransportType(connectorVal)
	}

	// Migration: if transport field is missing but deprecated connector is present, use connector value
	if !hasTransport && deprecatedConnector != "" {
		cfg.Connection.Transport = deprecatedConnector
	}

	return cfg
}
