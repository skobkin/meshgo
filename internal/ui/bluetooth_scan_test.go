package ui

import "testing"

func TestMergeBluetoothScanDevice(t *testing.T) {
	existing := DiscoveredBluetoothDevice{
		Name:                 "T",
		Address:              "AA:BB:CC:DD:EE:FF",
		RSSI:                 -80,
		HasMeshtasticService: false,
	}
	next := DiscoveredBluetoothDevice{
		Name:                 "T-Echo",
		Address:              "AA:BB:CC:DD:EE:FF",
		RSSI:                 -62,
		HasMeshtasticService: true,
	}

	merged := mergeBluetoothScanDevice(existing, next)
	if merged.Name != "T-Echo" {
		t.Fatalf("unexpected merged name: %q", merged.Name)
	}
	if merged.RSSI != -62 {
		t.Fatalf("unexpected merged RSSI: %d", merged.RSSI)
	}
	if !merged.HasMeshtasticService {
		t.Fatalf("expected meshtastic marker to be merged")
	}
}

func TestSortBluetoothScanDevices(t *testing.T) {
	devices := []DiscoveredBluetoothDevice{
		{Name: "Gamma", Address: "00:00:00:00:00:03", RSSI: -50, HasMeshtasticService: false},
		{Name: "Beta", Address: "00:00:00:00:00:02", RSSI: -90, HasMeshtasticService: true},
		{Name: "Alpha", Address: "00:00:00:00:00:01", RSSI: -60, HasMeshtasticService: true},
	}

	sortBluetoothScanDevices(devices)

	if devices[0].Name != "Alpha" {
		t.Fatalf("expected first device Alpha, got %q", devices[0].Name)
	}
	if devices[1].Name != "Beta" {
		t.Fatalf("expected second device Beta, got %q", devices[1].Name)
	}
	if devices[2].Name != "Gamma" {
		t.Fatalf("expected third device Gamma, got %q", devices[2].Name)
	}
}

func TestFormatBluetoothScanDevice(t *testing.T) {
	formatted := formatBluetoothScanDevice(DiscoveredBluetoothDevice{
		Name:                 "T-Echo",
		Address:              "AA:BB:CC:DD:EE:FF",
		RSSI:                 -63,
		HasMeshtasticService: true,
	})
	want := "T-Echo [Meshtastic]\nAA:BB:CC:DD:EE:FF RSSI: -63"
	if formatted != want {
		t.Fatalf("unexpected formatted value:\nwant: %q\ngot:  %q", want, formatted)
	}
}

func TestBluetoothScanDeviceAt(t *testing.T) {
	devices := []DiscoveredBluetoothDevice{{Address: "A"}, {Address: "B"}}

	if _, ok := bluetoothScanDeviceAt(devices, -1); ok {
		t.Fatalf("expected invalid index for -1")
	}
	if _, ok := bluetoothScanDeviceAt(devices, 2); ok {
		t.Fatalf("expected invalid index for 2")
	}

	device, ok := bluetoothScanDeviceAt(devices, 1)
	if !ok {
		t.Fatalf("expected valid index")
	}
	if device.Address != "B" {
		t.Fatalf("unexpected selected address: %q", device.Address)
	}
}
