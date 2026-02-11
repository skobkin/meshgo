package bluetoothutil

import (
	"fmt"
	"strings"

	"tinygo.org/x/bluetooth"
)

var (
	meshtasticServiceUUID   = MustParseUUID("6ba1b218-15a8-461f-9fa8-5dcae273eafd")
	meshtasticToRadioUUID   = MustParseUUID("f75c76d2-129e-4dad-a1dd-7866124401e7")
	meshtasticFromRadioUUID = MustParseUUID("2c55e69e-4993-11ed-b878-0242ac120002")
	meshtasticFromNumUUID   = MustParseUUID("ed9da18c-a800-4f66-a670-aa7547e34453")
)

func MustParseUUID(raw string) bluetooth.UUID {
	uuid, err := bluetooth.ParseUUID(strings.TrimSpace(raw))
	if err != nil {
		panic(fmt.Sprintf("invalid bluetooth UUID %q: %v", raw, err))
	}
	return uuid
}

func MeshtasticServiceUUID() bluetooth.UUID {
	return meshtasticServiceUUID
}

func MeshtasticToRadioUUID() bluetooth.UUID {
	return meshtasticToRadioUUID
}

func MeshtasticFromRadioUUID() bluetooth.UUID {
	return meshtasticFromRadioUUID
}

func MeshtasticFromNumUUID() bluetooth.UUID {
	return meshtasticFromNumUUID
}
