package transport

import (
	"context"
	"errors"
	"fmt"
	"math"
	"runtime"
	"strings"
	"sync"
	"time"

	"github.com/skobkin/meshgo/internal/bluetoothutil"
	"tinygo.org/x/bluetooth"
)

const (
	defaultBluetoothFrameQueueSize = 128
	defaultBluetoothReadBufferSize = 4096
	maxBluetoothDrainReads         = 256
	defaultBluetoothDiscoverWait   = 12 * time.Second
)

var (
	meshtasticServiceUUID   = mustParseBluetoothUUID("6ba1b218-15a8-461f-9fa8-5dcae273eafd")
	meshtasticToRadioUUID   = mustParseBluetoothUUID("f75c76d2-129e-4dad-a1dd-7866124401e7")
	meshtasticFromRadioUUID = mustParseBluetoothUUID("2c55e69e-4993-11ed-b878-0242ac120002")
	meshtasticFromNumUUID   = mustParseBluetoothUUID("ed9da18c-a800-4f66-a670-aa7547e34453")
)

type bluetoothConnState struct {
	device    bluetooth.Device
	toRadio   bluetooth.DeviceCharacteristic
	fromRadio bluetooth.DeviceCharacteristic
	fromNum   *bluetooth.DeviceCharacteristic

	frameCh  chan []byte
	drainReq chan struct{}
	closed   chan struct{}
}

type BluetoothTransport struct {
	address   string
	adapterID string

	mu      sync.RWMutex
	conn    *bluetoothConnState
	writeMu sync.Mutex
}

func NewBluetoothTransport(address, adapterID string) *BluetoothTransport {
	return &BluetoothTransport{
		address:   strings.TrimSpace(address),
		adapterID: strings.TrimSpace(adapterID),
	}
}

func (t *BluetoothTransport) Name() string {
	return "bluetooth"
}

func (t *BluetoothTransport) SetConfig(address, adapterID string) {
	t.mu.Lock()
	defer t.mu.Unlock()
	t.address = strings.TrimSpace(address)
	t.adapterID = strings.TrimSpace(adapterID)
}

func (t *BluetoothTransport) Address() string {
	t.mu.RLock()
	defer t.mu.RUnlock()
	return t.address
}

func (t *BluetoothTransport) AdapterID() string {
	t.mu.RLock()
	defer t.mu.RUnlock()
	return t.adapterID
}

func (t *BluetoothTransport) Connect(ctx context.Context) error {
	t.mu.Lock()
	defer t.mu.Unlock()

	if t.conn != nil {
		return nil
	}
	if err := ctx.Err(); err != nil {
		return err
	}
	if strings.TrimSpace(t.address) == "" {
		return errors.New("bluetooth address is empty")
	}

	addr, err := parseBluetoothAddress(t.address)
	if err != nil {
		return err
	}

	adapter := resolveBluetoothAdapter(t.adapterID)
	if err := adapter.Enable(); err != nil {
		return fmt.Errorf("enable bluetooth adapter: %w", err)
	}
	if err := ctx.Err(); err != nil {
		return err
	}

	device, err := adapter.Connect(addr, bluetooth.ConnectionParams{})
	if err != nil && shouldRetryBluetoothConnectWithDiscovery(err) {
		if discoverErr := discoverBluetoothDevice(ctx, adapter, addr); discoverErr != nil {
			return fmt.Errorf("connect bluetooth device %q: %w", t.address, errors.Join(err, fmt.Errorf("discovery failed: %w", discoverErr)))
		}
		device, err = adapter.Connect(addr, bluetooth.ConnectionParams{})
	}
	if err != nil {
		return fmt.Errorf("connect bluetooth device %q: %w", t.address, err)
	}

	services, err := device.DiscoverServices([]bluetooth.UUID{meshtasticServiceUUID})
	if err != nil {
		_ = device.Disconnect()
		return fmt.Errorf("discover meshtastic service: %w", err)
	}
	if len(services) == 0 {
		_ = device.Disconnect()
		return errors.New("meshtastic BLE service is not available")
	}

	chars, err := services[0].DiscoverCharacteristics([]bluetooth.UUID{
		meshtasticToRadioUUID,
		meshtasticFromRadioUUID,
		meshtasticFromNumUUID,
	})
	if err != nil {
		_ = device.Disconnect()
		return fmt.Errorf("discover meshtastic characteristics: %w", err)
	}
	if len(chars) != 3 {
		_ = device.Disconnect()
		return fmt.Errorf("unexpected characteristic count: %d", len(chars))
	}

	toRadio := chars[0]
	fromRadio := chars[1]
	fromNum := chars[2]

	state := &bluetoothConnState{
		device:    device,
		toRadio:   toRadio,
		fromRadio: fromRadio,
		fromNum:   &fromNum,
		frameCh:   make(chan []byte, defaultBluetoothFrameQueueSize),
		drainReq:  make(chan struct{}, 1),
		closed:    make(chan struct{}),
	}

	if err := state.fromNum.EnableNotifications(func(_ []byte) {
		t.requestDrain(state)
	}); err != nil {
		_ = device.Disconnect()
		return fmt.Errorf("subscribe to FromNum notifications: %w", err)
	}

	go t.runDrainLoop(state)
	t.requestDrain(state)

	if err := ctx.Err(); err != nil {
		_ = state.fromNum.EnableNotifications(nil)
		_ = device.Disconnect()
		return err
	}

	t.conn = state
	return nil
}

func (t *BluetoothTransport) Close() error {
	t.mu.Lock()
	state := t.conn
	t.conn = nil
	t.mu.Unlock()
	if state == nil {
		return nil
	}

	close(state.closed)

	var closeErr error
	if state.fromNum != nil {
		if err := state.fromNum.EnableNotifications(nil); err != nil {
			closeErr = errors.Join(closeErr, fmt.Errorf("disable FromNum notifications: %w", err))
		}
	}
	if err := state.device.Disconnect(); err != nil {
		closeErr = errors.Join(closeErr, fmt.Errorf("disconnect bluetooth device: %w", err))
	}

	return closeErr
}

func (t *BluetoothTransport) ReadFrame(ctx context.Context) ([]byte, error) {
	state, err := t.currentState()
	if err != nil {
		return nil, err
	}

	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	case <-state.closed:
		return nil, errors.New("transport is closed")
	case payload := <-state.frameCh:
		return payload, nil
	}
}

func (t *BluetoothTransport) WriteFrame(ctx context.Context, payload []byte) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if len(payload) > math.MaxUint16 {
		return fmt.Errorf("payload too large: %d", len(payload))
	}

	state, err := t.currentState()
	if err != nil {
		return err
	}

	t.writeMu.Lock()
	defer t.writeMu.Unlock()

	if err := ctx.Err(); err != nil {
		return err
	}

	written, err := state.toRadio.WriteWithoutResponse(payload)
	if err != nil {
		return fmt.Errorf("write to ToRadio: %w", err)
	}
	if written != len(payload) {
		return fmt.Errorf("short write to ToRadio: wrote %d of %d", written, len(payload))
	}

	return nil
}

func (t *BluetoothTransport) currentState() (*bluetoothConnState, error) {
	t.mu.RLock()
	defer t.mu.RUnlock()
	if t.conn == nil {
		return nil, errors.New("transport is not connected")
	}
	return t.conn, nil
}

func (t *BluetoothTransport) requestDrain(state *bluetoothConnState) {
	select {
	case <-state.closed:
		return
	default:
	}

	select {
	case state.drainReq <- struct{}{}:
	default:
	}
}

func (t *BluetoothTransport) runDrainLoop(state *bluetoothConnState) {
	for {
		select {
		case <-state.closed:
			return
		case <-state.drainReq:
			if err := t.drainFromRadio(state); err != nil {
				return
			}
		}
	}
}

func (t *BluetoothTransport) drainFromRadio(state *bluetoothConnState) error {
	for i := 0; i < maxBluetoothDrainReads; i++ {
		payload, err := readBluetoothCharacteristic(state.fromRadio, defaultBluetoothReadBufferSize)
		if err != nil {
			return fmt.Errorf("read FromRadio: %w", err)
		}
		if len(payload) == 0 {
			return nil
		}
		t.enqueueFrame(state, payload)
	}

	return fmt.Errorf("from-radio drain exceeded %d reads", maxBluetoothDrainReads)
}

func (t *BluetoothTransport) enqueueFrame(state *bluetoothConnState, payload []byte) {
	frame := append([]byte(nil), payload...)

	select {
	case <-state.closed:
		return
	default:
	}

	select {
	case state.frameCh <- frame:
	default:
		select {
		case <-state.frameCh:
		default:
		}
		select {
		case state.frameCh <- frame:
		default:
		}
	}
}

func readBluetoothCharacteristic(char bluetooth.DeviceCharacteristic, bufferSize int) ([]byte, error) {
	if bufferSize <= 0 {
		return nil, errors.New("buffer size must be positive")
	}

	buf := make([]byte, bufferSize)
	n, err := char.Read(buf)
	if err != nil {
		return nil, err
	}
	if n <= 0 {
		return nil, nil
	}
	if n > len(buf) {
		return nil, fmt.Errorf("payload length %d exceeds buffer size %d", n, len(buf))
	}

	payload := make([]byte, n)
	copy(payload, buf[:n])
	return payload, nil
}

func parseBluetoothAddress(raw string) (bluetooth.Address, error) {
	trimmed := strings.TrimSpace(raw)
	if trimmed == "" {
		return bluetooth.Address{}, errors.New("bluetooth address is empty")
	}

	mac, err := bluetooth.ParseMAC(strings.ToUpper(trimmed))
	if err != nil {
		return bluetooth.Address{}, fmt.Errorf("invalid bluetooth address %q: %w", trimmed, err)
	}

	return bluetooth.Address{MACAddress: bluetooth.MACAddress{MAC: mac}}, nil
}

func resolveBluetoothAdapter(adapterID string) *bluetooth.Adapter {
	trimmed := strings.TrimSpace(adapterID)
	if trimmed == "" {
		return bluetooth.DefaultAdapter
	}
	return bluetooth.NewAdapter(trimmed)
}

func mustParseBluetoothUUID(raw string) bluetooth.UUID {
	uuid, err := bluetooth.ParseUUID(strings.TrimSpace(raw))
	if err != nil {
		panic(fmt.Sprintf("invalid bluetooth UUID %q: %v", raw, err))
	}
	return uuid
}

func shouldRetryBluetoothConnectWithDiscovery(err error) bool {
	if err == nil || runtime.GOOS != "linux" {
		return false
	}
	msg := strings.ToLower(err.Error())
	if bluetoothutil.IsDBusErrorName(err, "org.freedesktop.DBus.Error.UnknownMethod") {
		return strings.Contains(msg, "org.freedesktop.dbus.properties") &&
			strings.Contains(msg, "method \"get\"")
	}

	return strings.Contains(msg, "org.freedesktop.dbus.properties") &&
		strings.Contains(msg, "method \"get\"") &&
		strings.Contains(msg, "doesn't exist")
}

func discoverBluetoothDevice(ctx context.Context, adapter *bluetooth.Adapter, target bluetooth.Address) error {
	if err := adapter.StopScan(); err != nil && !bluetoothutil.IsBenignStopScanError(err) {
		return fmt.Errorf("reset bluetooth scan state: %w", err)
	}

	scanCtx := ctx
	if _, hasDeadline := scanCtx.Deadline(); !hasDeadline {
		var cancel context.CancelFunc
		scanCtx, cancel = context.WithTimeout(scanCtx, defaultBluetoothDiscoverWait)
		defer cancel()
	}

	foundCh := make(chan struct{}, 1)
	scanErrCh := make(chan error, 1)
	go func() {
		scanErrCh <- adapter.Scan(func(_ *bluetooth.Adapter, result bluetooth.ScanResult) {
			if result.Address.MAC != target.MAC {
				return
			}
			select {
			case foundCh <- struct{}{}:
			default:
			}
			_ = adapter.StopScan()
		})
	}()

	found := false
	select {
	case <-foundCh:
		found = true
	case <-scanCtx.Done():
		_ = adapter.StopScan()
	}

	scanErr := <-scanErrCh
	if scanErr != nil && !bluetoothutil.IsBenignStopScanError(scanErr) {
		return fmt.Errorf("scan bluetooth devices: %w", scanErr)
	}

	if !found {
		return fmt.Errorf("device %q was not discovered; pair it in OS Bluetooth settings and keep it nearby", target.String())
	}

	return nil
}
