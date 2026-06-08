package textutil

import (
	"strings"
	"testing"
	"unicode/utf8"
)

func TestOptimizeUTF8StringWithHomoglyphsExactCyrillicBlock(t *testing.T) {
	for current := rune(0x0400); current <= 0x04ff; current++ {
		got := OptimizeUTF8StringWithHomoglyphs(string(current))
		replacement, mapped := cyrillicHomoglyphs[current]
		if mapped {
			if got != string(replacement) {
				t.Fatalf("U+%04X: expected %q, got %q", current, string(replacement), got)
			}

			continue
		}
		if got != string(current) {
			t.Fatalf("U+%04X: unmapped rune changed from %q to %q", current, string(current), got)
		}
	}
}

func TestOptimizeUTF8StringWithHomoglyphsCases(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  string
	}{
		{name: "empty"},
		{name: "ascii", input: "MeshGo 123", want: "MeshGo 123"},
		{name: "all mapped", input: "ЅІЈАВЕЗКМНОРСТХаеорсухѕіјҮ", want: "SIJABE3KMHOPCTXaeopcyxsijY"},
		{name: "unmapped Cyrillic", input: "БГДЁЖИЙЛПФЦЧШЩЪЫЬЭЮЯбгдёжзийклмнпфцчшщъыьэюя", want: "БГДЁЖИЙЛПФЦЧШЩЪЫЬЭЮЯбгдёжзийклмнпфцчшщъыьэюя"},
		{name: "other scripts", input: "Latin العربية 日本語", want: "Latin العربية 日本語"},
		{name: "punctuation whitespace controls", input: " А,\tа!\n\u0000 ", want: " A,\ta!\n\u0000 "},
		{name: "combining and normalization", input: "е\u0301 ё é e\u0301", want: "e\u0301 ё é e\u0301"},
		{name: "emoji and selectors", input: "А🙂 ❤️ 👨‍👩‍👧‍👦", want: "A🙂 ❤️ 👨‍👩‍👧‍👦"},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if got := OptimizeUTF8StringWithHomoglyphs(tc.input); got != tc.want {
				t.Fatalf("expected %q, got %q", tc.want, got)
			}
		})
	}
}

func TestOptimizeUTF8StringWithHomoglyphsInvariants(t *testing.T) {
	inputs := []string{
		"",
		"ASCII",
		"Привет, Meshtastic!",
		"Аа Бб 🙂 e\u0301 ❤️ 👨‍👩‍👧‍👦",
		string([]rune{0x0400, 0x0405, 0x0410, 0x04ae, 0x04ff}),
	}
	for _, input := range inputs {
		assertHomoglyphInvariants(t, input)
	}
}

func FuzzOptimizeUTF8StringWithHomoglyphs(f *testing.F) {
	for _, seed := range []string{
		"",
		"ASCII",
		"Привет, мир!",
		"АаЅѕІіЈјҮЗ",
		"العربية 日本語 🙂 ❤️ 👨‍👩‍👧‍👦 e\u0301",
	} {
		f.Add(seed)
	}

	f.Fuzz(func(t *testing.T, input string) {
		if !utf8.ValidString(input) {
			t.Skip()
		}
		assertHomoglyphInvariants(t, input)
	})
}

func assertHomoglyphInvariants(t *testing.T, input string) {
	t.Helper()
	output := OptimizeUTF8StringWithHomoglyphs(input)
	if !utf8.ValidString(output) {
		t.Fatalf("output is not valid UTF-8: %q", output)
	}
	if utf8.RuneCountInString(output) != utf8.RuneCountInString(input) {
		t.Fatalf("rune count changed: input=%d output=%d", utf8.RuneCountInString(input), utf8.RuneCountInString(output))
	}
	if len(output) > len(input) {
		t.Fatalf("byte length increased: input=%d output=%d", len(input), len(output))
	}
	if twice := OptimizeUTF8StringWithHomoglyphs(output); twice != output {
		t.Fatalf("transformation is not idempotent: once=%q twice=%q", output, twice)
	}

	inputRunes := []rune(input)
	outputRunes := []rune(output)
	for index, current := range inputRunes {
		replacement, mapped := cyrillicHomoglyphs[current]
		if mapped {
			if outputRunes[index] != replacement {
				t.Fatalf("mapped rune %q became %q instead of %q", current, outputRunes[index], replacement)
			}

			continue
		}
		if outputRunes[index] != current {
			t.Fatalf("unmapped rune %q changed to %q", current, outputRunes[index])
		}
	}

	if strings.ContainsRune(output, utf8.RuneError) && !strings.ContainsRune(input, utf8.RuneError) {
		t.Fatalf("transformation introduced replacement rune")
	}
}
