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
	defaultBluetoothSubscribeWait  = 8 * time.Second
)

type bluetoothConnState struct {
	device    bluetooth.Device
	toRadio   bluetooth.DeviceCharacteristic
	fromRadio bluetooth.DeviceCharacteristic
	fromNum   *bluetooth.DeviceCharacteristic

	frameCh  chan []byte
	drainReq chan struct{}
	closed   chan struct{}

	closeOnce sync.Once
	errMu     sync.RWMutex
	asyncErr  error
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

func (t *BluetoothTransport) StatusTarget() string {
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

	logger := transportLogger("bluetooth", "address", t.address, "adapter", t.adapterID)

	if t.conn != nil {
		logger.Debug("connect skipped: already connected")
		return nil
	}
	if err := ctx.Err(); err != nil {
		logger.Debug("connect canceled", "error", err)
		return err
	}
	if strings.TrimSpace(t.address) == "" {
		logger.Warn("connect failed: address is empty")
		return errors.New("bluetooth address is empty")
	}

	addr, err := parseBluetoothAddress(t.address)
	if err != nil {
		logger.Warn("connect failed: invalid address", "error", err)
		return err
	}

	adapter := bluetoothutil.ResolveAdapter(t.adapterID)
	logger.Info("connecting")
	logger.Debug("enabling adapter")
	if err := adapter.Enable(); err != nil {
		logger.Warn("enable adapter failed", "error", err)
		return fmt.Errorf("enable bluetooth adapter: %w", err)
	}
	logger.Debug("adapter enabled")
	if err := ctx.Err(); err != nil {
		logger.Debug("connect canceled after adapter enable", "error", err)
		return err
	}

	logger.Debug("connecting device")
	device, err := adapter.Connect(addr, bluetooth.ConnectionParams{})
	if err != nil && shouldRetryBluetoothConnectWithDiscovery(err) {
		logger.Info("direct connect failed, trying discovery fallback", "error", err)
		if discoverErr := discoverBluetoothDevice(ctx, adapter, addr); discoverErr != nil {
			logger.Warn("discovery fallback failed", "error", discoverErr)
			return fmt.Errorf("connect bluetooth device %q: %w", t.address, errors.Join(err, fmt.Errorf("discovery failed: %w", discoverErr)))
		}
		logger.Debug("retrying device connect after discovery")
		device, err = adapter.Connect(addr, bluetooth.ConnectionParams{})
	}
	if err != nil {
		logger.Warn("connect device failed", "error", err)
		return fmt.Errorf("connect bluetooth device %q: %w", t.address, err)
	}
	logger.Debug("device connected")

	logger.Debug("discovering meshtastic service")
	services, err := device.DiscoverServices([]bluetooth.UUID{bluetoothutil.MeshtasticServiceUUID()})
	if err != nil {
		_ = device.Disconnect()
		logger.Warn("discover service failed", "error", err)
		return fmt.Errorf("discover meshtastic service: %w", err)
	}
	if len(services) == 0 {
		_ = device.Disconnect()
		logger.Warn("meshtastic service is not available")
		return errors.New("meshtastic BLE service is not available")
	}
	logger.Debug("meshtastic service discovered", "count", len(services))

	logger.Debug("discovering meshtastic characteristics")
	chars, err := services[0].DiscoverCharacteristics([]bluetooth.UUID{
		bluetoothutil.MeshtasticToRadioUUID(),
		bluetoothutil.MeshtasticFromRadioUUID(),
		bluetoothutil.MeshtasticFromNumUUID(),
	})
	if err != nil {
		_ = device.Disconnect()
		logger.Warn("discover characteristics failed", "error", err)
		return fmt.Errorf("discover meshtastic characteristics: %w", err)
	}
	if len(chars) != 3 {
		_ = device.Disconnect()
		logger.Warn("unexpected characteristic count", "count", len(chars))
		return fmt.Errorf("unexpected characteristic count: %d", len(chars))
	}
	logger.Debug("meshtastic characteristics discovered")

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

	logger.Debug("subscribing to notifications")
	if err := enableBluetoothNotificationsWithTimeout(ctx, device, *state.fromNum, func(_ []byte) {
		t.requestDrain(state)
	}, defaultBluetoothSubscribeWait); err != nil {
		_ = device.Disconnect()
		logger.Warn("subscribe to notifications failed", "error", err)
		return fmt.Errorf("subscribe to FromNum notifications: %w", err)
	}
	logger.Debug("subscribed to notifications")

	go t.runDrainLoop(state)
	t.requestDrain(state)

	if err := ctx.Err(); err != nil {
		state.markClosed()
		_ = state.fromNum.EnableNotifications(nil)
		_ = device.Disconnect()
		logger.Debug("connect canceled after setup", "error", err)
		return err
	}

	t.conn = state
	logger.Info("connected")
	return nil
}

func (t *BluetoothTransport) Close() error {
	t.mu.Lock()
	logger := transportLogger("bluetooth", "address", t.address, "adapter", t.adapterID)
	state := t.conn
	t.conn = nil
	t.mu.Unlock()
	if state == nil {
		logger.Debug("close skipped: not connected")
		return nil
	}

	logger.Info("closing connection")
	state.markClosed()

	var closeErr error
	if state.fromNum != nil {
		if err := state.fromNum.EnableNotifications(nil); err != nil {
			closeErr = errors.Join(closeErr, fmt.Errorf("disable FromNum notifications: %w", err))
			logger.Warn("disable notifications failed", "error", err)
		}
	}
	if err := state.device.Disconnect(); err != nil {
		closeErr = errors.Join(closeErr, fmt.Errorf("disconnect bluetooth device: %w", err))
		logger.Warn("disconnect failed", "error", err)
	}

	if closeErr != nil {
		return closeErr
	}
	logger.Info("closed")

	return nil
}

func (t *BluetoothTransport) ReadFrame(ctx context.Context) ([]byte, error) {
	logger := transportLogger("bluetooth")
	state, err := t.currentState()
	if err != nil {
		logger.Debug("read frame failed: not connected", "error", err)
		return nil, err
	}

	select {
	case <-ctx.Done():
		logger.Debug("read frame canceled", "error", ctx.Err())
		return nil, ctx.Err()
	case <-state.closed:
		if err := state.closeErr(); err != nil {
			logger.Warn("read frame failed: connection closed with async error", "error", err)
			return nil, err
		}
		logger.Debug("read frame failed: transport closed")
		return nil, errors.New("transport is closed")
	case payload := <-state.frameCh:
		logger.Debug("read frame", "len", len(payload))
		return payload, nil
	}
}

func (t *BluetoothTransport) WriteFrame(ctx context.Context, payload []byte) error {
	logger := transportLogger("bluetooth")
	if err := ctx.Err(); err != nil {
		logger.Debug("write frame canceled", "error", err)
		return err
	}
	if len(payload) > math.MaxUint16 {
		logger.Warn("write frame failed: payload too large", "payload_len", len(payload))
		return fmt.Errorf("payload too large: %d", len(payload))
	}

	state, err := t.currentState()
	if err != nil {
		logger.Debug("write frame failed: not connected", "error", err)
		return err
	}

	t.writeMu.Lock()
	defer t.writeMu.Unlock()

	if err := ctx.Err(); err != nil {
		logger.Debug("write frame canceled", "error", err)
		return err
	}
	select {
	case <-state.closed:
		if err := state.closeErr(); err != nil {
			logger.Warn("write frame failed: connection closed with async error", "error", err)
			return err
		}
		logger.Debug("write frame failed: transport closed")
		return errors.New("transport is closed")
	default:
	}

	written, err := state.toRadio.WriteWithoutResponse(payload)
	if err != nil {
		logger.Warn("write frame failed", "payload_len", len(payload), "error", err)
		return fmt.Errorf("write to ToRadio: %w", err)
	}
	if written != len(payload) {
		logger.Warn("write frame failed: short write", "payload_len", len(payload), "written", written)
		return fmt.Errorf("short write to ToRadio: wrote %d of %d", written, len(payload))
	}
	logger.Debug("write frame", "payload_len", len(payload))

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
	logger := transportLogger("bluetooth")
	for {
		select {
		case <-state.closed:
			logger.Debug("drain loop stopped")
			return
		case <-state.drainReq:
			if err := t.drainFromRadio(state); err != nil {
				logger.Warn("drain failed", "error", err)
				t.failState(state, err)
				return
			}
		}
	}
}

func (t *BluetoothTransport) failState(state *bluetoothConnState, err error) {
	logger := transportLogger("bluetooth")
	state.setAsyncError(err)
	state.markClosed()

	t.mu.Lock()
	if t.conn == state {
		t.conn = nil
	}
	t.mu.Unlock()

	if state.fromNum != nil {
		_ = state.fromNum.EnableNotifications(nil)
	}
	_ = state.device.Disconnect()
	logger.Warn("connection failed and was closed", "error", err)
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
	logger := transportLogger("bluetooth")
	frame := append([]byte(nil), payload...)

	select {
	case <-state.closed:
		return
	default:
	}

	select {
	case state.frameCh <- frame:
	default:
		logger.Warn("frame queue full, dropping oldest frame", "capacity", cap(state.frameCh), "dropped_len", len(frame))
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
	logger := transportLogger("bluetooth", "target", target.String())
	logger.Info("starting device discovery fallback")
	if err := bluetoothutil.StopScan(adapter); err != nil {
		logger.Warn("failed to reset scan state before discovery", "error", err)
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
		logger.Info("target device discovered")
	case <-scanCtx.Done():
		logger.Warn("device discovery timed out or canceled", "error", scanCtx.Err())
		_ = bluetoothutil.StopScan(adapter)
	}

	scanErr := <-scanErrCh
	if scanErr = bluetoothutil.NormalizeScanError(scanErr); scanErr != nil {
		logger.Warn("device discovery scan failed", "error", scanErr)
		return fmt.Errorf("scan bluetooth devices: %w", scanErr)
	}

	if !found {
		logger.Warn("target device not discovered")
		return fmt.Errorf("device %q was not discovered; pair it in OS Bluetooth settings and keep it nearby", target.String())
	}

	logger.Info("device discovery completed")
	return nil
}

func (s *bluetoothConnState) markClosed() {
	s.closeOnce.Do(func() {
		close(s.closed)
	})
}

func (s *bluetoothConnState) setAsyncError(err error) {
	if err == nil {
		return
	}
	s.errMu.Lock()
	if s.asyncErr == nil {
		s.asyncErr = err
	}
	s.errMu.Unlock()
}

func (s *bluetoothConnState) closeErr() error {
	s.errMu.RLock()
	defer s.errMu.RUnlock()
	return s.asyncErr
}

func enableBluetoothNotificationsWithTimeout(
	ctx context.Context,
	device bluetooth.Device,
	char bluetooth.DeviceCharacteristic,
	callback func([]byte),
	wait time.Duration,
) error {
	if wait <= 0 {
		wait = defaultBluetoothSubscribeWait
	}

	done := make(chan error, 1)
	go func() {
		done <- char.EnableNotifications(callback)
	}()

	timer := time.NewTimer(wait)
	defer timer.Stop()

	select {
	case err := <-done:
		return err
	case <-ctx.Done():
		_ = device.Disconnect()
		select {
		case <-done:
		case <-time.After(2 * time.Second):
		}
		return ctx.Err()
	case <-timer.C:
		_ = device.Disconnect()
		select {
		case err := <-done:
			if err != nil {
				return fmt.Errorf("timed out after %s (abort returned: %w)", wait, err)
			}
		case <-time.After(2 * time.Second):
		}
		return fmt.Errorf("timed out after %s", wait)
	}
}
