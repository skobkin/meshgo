package main

import "testing"

func TestParseLaunchOptions(t *testing.T) {
	tests := []struct {
		name    string
		args    []string
		want    launchOptions
		wantErr bool
	}{
		{name: "defaults", args: nil, want: launchOptions{StartHidden: false}},
		{name: "start hidden", args: []string{"--start-hidden"}, want: launchOptions{StartHidden: true}},
		{name: "unexpected positional", args: []string{"extra"}, wantErr: true},
		{name: "unknown flag", args: []string{"--nope"}, wantErr: true},
	}

	for _, tc := range tests {
		got, err := parseLaunchOptions(tc.args)
		if tc.wantErr {
			if err == nil {
				t.Fatalf("%s: expected error, got nil", tc.name)
			}

			continue
		}
		if err != nil {
			t.Fatalf("%s: unexpected error: %v", tc.name, err)
		}
		if got != tc.want {
			t.Fatalf("%s: expected %+v, got %+v", tc.name, tc.want, got)
		}
	}
}
