package app

// NodeSettingsTarget identifies which node should be read/modified.
// IsLocal is reserved for future remote editing support.
type NodeSettingsTarget struct {
	NodeID  string
	IsLocal bool
}

// NodeUserSettings contains editable owner/user settings.
type NodeUserSettings struct {
	NodeID          string
	LongName        string
	ShortName       string
	HamLicensed     bool
	IsUnmessageable bool
}

// NodeSecuritySettings contains editable security config settings.
type NodeSecuritySettings struct {
	NodeID    string
	PublicKey []byte
	//nolint:gosec // Private key bytes are intentionally modeled for device settings read/write.
	PrivateKey          []byte
	AdminKeys           [][]byte
	IsManaged           bool
	SerialEnabled       bool
	DebugLogAPIEnabled  bool
	AdminChannelEnabled bool
}

// NodeLoRaSettings contains editable LoRa config settings.
type NodeLoRaSettings struct {
	NodeID              string
	UsePreset           bool
	ModemPreset         int32
	Bandwidth           uint32
	SpreadFactor        uint32
	CodingRate          uint32
	FrequencyOffset     float32
	Region              int32
	HopLimit            uint32
	TxEnabled           bool
	TxPower             int32
	ChannelNum          uint32
	OverrideDutyCycle   bool
	Sx126XRxBoostedGain bool
	OverrideFrequency   float32
	PaFanDisabled       bool
	IgnoreIncoming      []uint32
	IgnoreMqtt          bool
	ConfigOkToMqtt      bool
}

// NodeDeviceSettings contains editable device config settings.
type NodeDeviceSettings struct {
	NodeID                 string
	Role                   int32
	ButtonGPIO             uint32
	BuzzerGPIO             uint32
	RebroadcastMode        int32
	NodeInfoBroadcastSecs  uint32
	DoubleTapAsButtonPress bool
	DisableTripleClick     bool
	Tzdef                  string
	LedHeartbeatDisabled   bool
	BuzzerMode             int32
}

// NodePositionSettings contains editable position config settings.
type NodePositionSettings struct {
	NodeID                            string
	PositionBroadcastSecs             uint32
	PositionBroadcastSmartEnabled     bool
	FixedPosition                     bool
	FixedLatitude                     *float64
	FixedLongitude                    *float64
	FixedAltitude                     *int32
	RemoveFixedPosition               bool
	GpsUpdateInterval                 uint32
	PositionFlags                     uint32
	RxGPIO                            uint32
	TxGPIO                            uint32
	BroadcastSmartMinimumDistance     uint32
	BroadcastSmartMinimumIntervalSecs uint32
	GpsEnGPIO                         uint32
	GpsMode                           int32
}

// NodePowerSettings contains editable power config settings.
type NodePowerSettings struct {
	NodeID                     string
	IsPowerSaving              bool
	OnBatteryShutdownAfterSecs uint32
	AdcMultiplierOverride      float32
	WaitBluetoothSecs          uint32
	SdsSecs                    uint32
	LsSecs                     uint32
	MinWakeSecs                uint32
	DeviceBatteryInaAddress    uint32
	PowermonEnables            uint64
}

// NodeDisplaySettings contains editable display config settings.
type NodeDisplaySettings struct {
	NodeID                 string
	ScreenOnSecs           uint32
	AutoScreenCarouselSecs uint32
	CompassNorthTop        bool
	FlipScreen             bool
	Units                  int32
	Oled                   int32
	DisplayMode            int32
	HeadingBold            bool
	WakeOnTapOrMotion      bool
	CompassOrientation     int32
	Use12HClock            bool
}

// NodeBluetoothSettings contains editable Bluetooth config settings.
type NodeBluetoothSettings struct {
	NodeID   string
	Enabled  bool
	Mode     int32
	FixedPIN uint32
}

// NodeNetworkSettings contains editable network config settings.
type NodeNetworkSettings struct {
	NodeID           string
	WifiEnabled      bool
	WifiSSID         string
	WifiPSK          string
	NTPServer        string
	EthernetEnabled  bool
	AddressMode      int32
	IPv4Address      uint32
	IPv4Gateway      uint32
	IPv4Subnet       uint32
	IPv4DNS          uint32
	RsyslogServer    string
	EnabledProtocols uint32
	IPv6Enabled      bool
}

// NodeMQTTSettings contains editable MQTT module settings.
type NodeMQTTSettings struct {
	NodeID   string
	Enabled  bool
	Address  string
	Username string
	//nolint:gosec // Password is intentionally modeled for MQTT settings read/write.
	Password                      string
	EncryptionEnabled             bool
	JSONEnabled                   bool
	TLSEnabled                    bool
	Root                          string
	ProxyToClientEnabled          bool
	MapReportingEnabled           bool
	MapReportPublishIntervalSecs  uint32
	MapReportPositionPrecision    uint32
	MapReportShouldReportLocation bool
}

// NodeRangeTestSettings contains editable Range Test module settings.
type NodeRangeTestSettings struct {
	NodeID string
	// ClearOnReboot is intentionally tracked even though not exposed in UI yet,
	// so save operations preserve device-side value for hidden fields.
	ClearOnReboot bool
	Enabled       bool
	Sender        uint32
	Save          bool
}

// NodeSerialSettings contains editable Serial module settings.
type NodeSerialSettings struct {
	NodeID                    string
	Enabled                   bool
	EchoEnabled               bool
	RXGPIO                    uint32
	TXGPIO                    uint32
	Baud                      int32
	Timeout                   uint32
	Mode                      int32
	OverrideConsoleSerialPort bool
}

// NodeExternalNotificationSettings contains editable External Notification module settings.
type NodeExternalNotificationSettings struct {
	NodeID             string
	Enabled            bool
	OutputMS           uint32
	OutputGPIO         uint32
	OutputVibraGPIO    uint32
	OutputBuzzerGPIO   uint32
	OutputActiveHigh   bool
	AlertMessageLED    bool
	AlertMessageVibra  bool
	AlertMessageBuzzer bool
	AlertBellLED       bool
	AlertBellVibra     bool
	AlertBellBuzzer    bool
	UsePWMBuzzer       bool
	NagTimeoutSecs     uint32
	Ringtone           string
	UseI2SAsBuzzer     bool
}

// NodeStoreForwardSettings contains editable Store & Forward module settings.
type NodeStoreForwardSettings struct {
	NodeID              string
	Enabled             bool
	Heartbeat           bool
	Records             uint32
	HistoryReturnMax    uint32
	HistoryReturnWindow uint32
	IsServer            bool
}

// NodeTelemetrySettings contains editable Telemetry module settings.
type NodeTelemetrySettings struct {
	NodeID                        string
	DeviceUpdateInterval          uint32
	EnvironmentUpdateInterval     uint32
	EnvironmentMeasurementEnabled bool
	EnvironmentScreenEnabled      bool
	EnvironmentDisplayFahrenheit  bool
	AirQualityEnabled             bool
	AirQualityInterval            uint32
	PowerMeasurementEnabled       bool
	PowerUpdateInterval           uint32
	PowerScreenEnabled            bool
	HealthMeasurementEnabled      bool
	HealthUpdateInterval          uint32
	HealthScreenEnabled           bool
	DeviceTelemetryEnabled        bool
	AirQualityScreenEnabled       bool
}

// NodeCannedMessageSettings contains editable Canned Message module settings.
type NodeCannedMessageSettings struct {
	NodeID                string
	Rotary1Enabled        bool
	InputBrokerPinA       uint32
	InputBrokerPinB       uint32
	InputBrokerPinPress   uint32
	InputBrokerEventCW    int32
	InputBrokerEventCCW   int32
	InputBrokerEventPress int32
	UpDown1Enabled        bool
	Enabled               bool
	AllowInputSource      string
	SendBell              bool
	Messages              string
}

// NodeAudioSettings contains editable Audio module settings.
type NodeAudioSettings struct {
	NodeID        string
	Codec2Enabled bool
	PTTPin        uint32
	Bitrate       int32
	I2SWordSelect uint32
	I2SDataIn     uint32
	I2SDataOut    uint32
	I2SClock      uint32
}

// NodeRemoteHardwareSettings contains editable Remote Hardware module settings.
type NodeRemoteHardwareSettings struct {
	NodeID                  string
	Enabled                 bool
	AllowUndefinedPinAccess bool
	AvailablePins           []uint32
}

// NodeNeighborInfoSettings contains editable Neighbor Info module settings.
type NodeNeighborInfoSettings struct {
	NodeID             string
	Enabled            bool
	UpdateIntervalSecs uint32
	TransmitOverLoRa   bool
}

// NodeAmbientLightingSettings contains editable Ambient Lighting module settings.
type NodeAmbientLightingSettings struct {
	NodeID   string
	LEDState bool
	Current  uint32
	Red      uint32
	Green    uint32
	Blue     uint32
}

// NodeDetectionSensorSettings contains editable Detection Sensor module settings.
type NodeDetectionSensorSettings struct {
	NodeID               string
	Enabled              bool
	MinimumBroadcastSecs uint32
	StateBroadcastSecs   uint32
	SendBell             bool
	Name                 string
	MonitorPin           uint32
	DetectionTriggerType int32
	UsePullup            bool
}

// NodePaxcounterSettings contains editable Paxcounter module settings.
type NodePaxcounterSettings struct {
	NodeID             string
	Enabled            bool
	UpdateIntervalSecs uint32
	WifiThreshold      int32
	BLEThreshold       int32
}

// NodeStatusMessageSettings contains editable Status Message module settings.
type NodeStatusMessageSettings struct {
	NodeID     string
	NodeStatus string
}

// NodeChannelSettings contains editable settings for one channel row.
type NodeChannelSettings struct {
	Name              string
	PSK               []byte
	ID                uint32
	UplinkEnabled     bool
	DownlinkEnabled   bool
	PositionPrecision uint32
	Muted             bool
}

// NodeChannelSettingsList contains editable channel list settings.
type NodeChannelSettingsList struct {
	NodeID   string
	MaxSlots int
	Channels []NodeChannelSettings
}
