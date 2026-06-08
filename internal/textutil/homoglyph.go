package textutil

import "strings"

var cyrillicHomoglyphs = map[rune]rune{
	'\u0405': 'S',
	'\u0406': 'I',
	'\u0408': 'J',
	'\u0410': 'A',
	'\u0412': 'B',
	'\u0415': 'E',
	'\u0417': '3',
	'\u041a': 'K',
	'\u041c': 'M',
	'\u041d': 'H',
	'\u041e': 'O',
	'\u0420': 'P',
	'\u0421': 'C',
	'\u0422': 'T',
	'\u0425': 'X',
	'\u0430': 'a',
	'\u0435': 'e',
	'\u043e': 'o',
	'\u0440': 'p',
	'\u0441': 'c',
	'\u0443': 'y',
	'\u0445': 'x',
	'\u0455': 's',
	'\u0456': 'i',
	'\u0458': 'j',
	'\u04ae': 'Y',
}

// OptimizeUTF8StringWithHomoglyphs replaces the conservative Cyrillic
// homoglyph set used by Meshtastic Android with single-byte ASCII runes.
func OptimizeUTF8StringWithHomoglyphs(input string) string {
	var output strings.Builder
	output.Grow(len(input))
	for _, current := range input {
		if replacement, ok := cyrillicHomoglyphs[current]; ok {
			output.WriteRune(replacement)

			continue
		}
		output.WriteRune(current)
	}

	return output.String()
}
