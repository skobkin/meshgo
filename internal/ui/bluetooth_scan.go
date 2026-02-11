package ui

import (
	"context"
	"errors"
	"fmt"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/skobkin/meshgo/internal/bluetoothutil"
	"tinygo.org/x/bluetooth"
)

const defaultBluetoothScanDuration = 10 * time.Second

// DiscoveredBluetoothDevice represents a discovered BLE device entry for UI selection.
type DiscoveredBluetoothDevice struct {
	Name                 string
	Address              string
	RSSI                 int
	HasMeshtasticService bool
}

// BluetoothScanner scans nearby BLE devices and returns normalized results.
type BluetoothScanner interface {
	Scan(ctx context.Context, adapterID string) ([]DiscoveredBluetoothDevice, error)
}

// TinyGoBluetoothScanner is a BluetoothScanner backed by tinygo.org/x/bluetooth.
type TinyGoBluetoothScanner struct {
	scanDuration time.Duration
	mu           sync.Mutex
}

func NewTinyGoBluetoothScanner(scanDuration time.Duration) *TinyGoBluetoothScanner {
	if scanDuration <= 0 {
		scanDuration = defaultBluetoothScanDuration
	}

	return &TinyGoBluetoothScanner{scanDuration: scanDuration}
}

func (s *TinyGoBluetoothScanner) Scan(ctx context.Context, adapterID string) ([]DiscoveredBluetoothDevice, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	adapter := bluetoothutil.ResolveAdapter(adapterID)
	if err := bluetoothutil.EnableAdapter(adapter); err != nil {
		return nil, fmt.Errorf("enable bluetooth adapter: %w", err)
	}
	if err := bluetoothutil.StopScan(adapter); err != nil {
		return nil, fmt.Errorf("reset bluetooth scan state: %w", err)
	}

	scanCtx := ctx
	if _, hasDeadline := scanCtx.Deadline(); !hasDeadline {
		var cancel context.CancelFunc
		scanCtx, cancel = context.WithTimeout(scanCtx, s.scanDuration)
		defer cancel()
	}

	var (
		mu      sync.Mutex
		devices = make(map[string]DiscoveredBluetoothDevice)
	)
	scanErrCh := make(chan error, 1)

	go func() {
		scanErrCh <- runBluetoothScan(adapter, func(_ *bluetooth.Adapter, result bluetooth.ScanResult) {
			entry := scanDeviceFromResult(result)
			if entry.Address == "" {
				return
			}

			mu.Lock()
			defer mu.Unlock()

			if existing, ok := devices[entry.Address]; ok {
				devices[entry.Address] = mergeBluetoothScanDevice(existing, entry)

				return
			}
			devices[entry.Address] = entry
		})
	}()

	if err := awaitScanCompletion(scanCtx, adapter, scanErrCh); err != nil {
		return nil, err
	}

	mu.Lock()
	result := make([]DiscoveredBluetoothDevice, 0, len(devices))
	for _, device := range devices {
		result = append(result, device)
	}
	mu.Unlock()

	sortBluetoothScanDevices(result)

	return result, nil
}

func runBluetoothScan(adapter *bluetooth.Adapter, callback func(*bluetooth.Adapter, bluetooth.ScanResult)) error {
	var lastErr error
	for attempt := 0; attempt < 2; attempt++ {
		err := adapter.Scan(callback)
		if err == nil {
			return nil
		}
		lastErr = err
		if !bluetoothutil.IsScanAlreadyInProgressError(err) {
			return err
		}
		if stopErr := bluetoothutil.StopScan(adapter); stopErr != nil {
			return errors.Join(err, fmt.Errorf("stop stale bluetooth scan: %w", stopErr))
		}
	}

	return lastErr
}

func awaitScanCompletion(ctx context.Context, adapter *bluetooth.Adapter, scanErrCh <-chan error) error {
	select {
	case err := <-scanErrCh:
		if err = bluetoothutil.NormalizeScanError(err); err != nil {
			return fmt.Errorf("scan bluetooth devices: %w", err)
		}

		return nil
	case <-ctx.Done():
		if err := bluetoothutil.StopScan(adapter); err != nil {
			return fmt.Errorf("stop bluetooth scan: %w", err)
		}
		err := <-scanErrCh
		if err = bluetoothutil.NormalizeScanError(err); err != nil {
			return fmt.Errorf("scan bluetooth devices: %w", err)
		}
		if errors.Is(ctx.Err(), context.DeadlineExceeded) {
			return nil
		}

		return ctx.Err()
	}
}

func scanDeviceFromResult(result bluetooth.ScanResult) DiscoveredBluetoothDevice {
	return DiscoveredBluetoothDevice{
		Name:                 strings.TrimSpace(result.LocalName()),
		Address:              normalizeBluetoothAddress(result.Address.String()),
		RSSI:                 int(result.RSSI),
		HasMeshtasticService: result.HasServiceUUID(bluetoothutil.MeshtasticServiceUUID()),
	}
}

func mergeBluetoothScanDevice(existing, next DiscoveredBluetoothDevice) DiscoveredBluetoothDevice {
	merged := existing

	if len(strings.TrimSpace(next.Name)) > len(strings.TrimSpace(merged.Name)) {
		merged.Name = next.Name
	}
	if next.RSSI > merged.RSSI {
		merged.RSSI = next.RSSI
	}
	merged.HasMeshtasticService = merged.HasMeshtasticService || next.HasMeshtasticService

	return merged
}

func sortBluetoothScanDevices(devices []DiscoveredBluetoothDevice) {
	sort.Slice(devices, func(i, j int) bool {
		if devices[i].HasMeshtasticService != devices[j].HasMeshtasticService {
			return devices[i].HasMeshtasticService
		}
		if devices[i].RSSI != devices[j].RSSI {
			return devices[i].RSSI > devices[j].RSSI
		}

		leftName := strings.ToLower(strings.TrimSpace(devices[i].Name))
		rightName := strings.ToLower(strings.TrimSpace(devices[j].Name))
		if leftName != rightName {
			return leftName < rightName
		}

		return devices[i].Address < devices[j].Address
	})
}

func bluetoothScanDeviceTitle(device DiscoveredBluetoothDevice) string {
	displayName := strings.TrimSpace(device.Name)
	if displayName == "" {
		displayName = "(unnamed)"
	}

	marker := ""
	if device.HasMeshtasticService {
		marker = " [Meshtastic]"
	}

	return fmt.Sprintf("%s%s", displayName, marker)
}

func bluetoothScanDeviceDetails(device DiscoveredBluetoothDevice) string {
	return fmt.Sprintf("%s RSSI: %d", device.Address, device.RSSI)
}

func formatBluetoothScanDevice(device DiscoveredBluetoothDevice) string {
	return fmt.Sprintf("%s\n%s", bluetoothScanDeviceTitle(device), bluetoothScanDeviceDetails(device))
}

func bluetoothScanDeviceAt(devices []DiscoveredBluetoothDevice, index int) (DiscoveredBluetoothDevice, bool) {
	if index < 0 || index >= len(devices) {
		return DiscoveredBluetoothDevice{}, false
	}

	return devices[index], true
}

func normalizeBluetoothAddress(address string) string {
	return strings.ToUpper(strings.TrimSpace(address))
}
