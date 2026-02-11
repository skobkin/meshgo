package platform

var windowsBluetoothSettingsCommands = []commandSpec{
	{name: "cmd", args: []string{"/c", "start", "", "ms-settings:bluetooth"}},
}
