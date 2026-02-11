package bluetoothutil

import "testing"

func TestMeshtasticUUIDsAreDefinedAndDistinct(t *testing.T) {
	service := MeshtasticServiceUUID()
	toRadio := MeshtasticToRadioUUID()
	fromRadio := MeshtasticFromRadioUUID()
	fromNum := MeshtasticFromNumUUID()

	if service == toRadio || service == fromRadio || service == fromNum {
		t.Fatalf("service UUID must be distinct from characteristic UUIDs")
	}
	if toRadio == fromRadio || toRadio == fromNum || fromRadio == fromNum {
		t.Fatalf("meshtastic characteristic UUIDs must be distinct")
	}
}

func TestMustParseUUIDPanicsOnInvalidValue(t *testing.T) {
	defer func() {
		if recover() == nil {
			t.Fatalf("expected panic for invalid UUID")
		}
	}()
	_ = mustParseUUID("not-a-uuid")
}
