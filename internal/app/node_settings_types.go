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
	NodeID              string
	PublicKey           []byte
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
