package platform

var linuxBluetoothSettingsCommands = []commandSpec{
	{name: "systemsettings6", args: []string{"kcm_bluetooth"}},
	{name: "systemsettings5", args: []string{"kcm_bluetooth"}},
	{name: "systemsettings", args: []string{"kcm_bluetooth"}},
	{name: "gnome-control-center", args: []string{"bluetooth"}},
	{name: "kcmshell6", args: []string{"kcm_bluetooth"}},
	{name: "kcmshell5", args: []string{"kcm_bluetooth"}},
	{name: "kcmshell4", args: []string{"bluetooth"}},
	{name: "blueman-manager"},
	{name: "xdg-open", args: []string{"bluetooth://"}},
}
