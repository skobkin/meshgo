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
