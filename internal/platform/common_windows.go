//go:build windows

package platform

import "strings"

func buildWindowsCommandLine(executable string, args []string) string {
	fields := make([]string, 0, 1+len(args))
	fields = append(fields, quoteWindowsCommandLineArg(executable))
	for _, arg := range args {
		fields = append(fields, quoteWindowsCommandLineArg(arg))
	}
	return strings.Join(fields, " ")
}

func quoteWindowsCommandLineArg(arg string) string {
	if arg == "" {
		return `""`
	}
	if !strings.ContainsAny(arg, " \t\n\v\"") {
		return arg
	}

	var b strings.Builder
	b.WriteByte('"')
	backslashes := 0
	for i := 0; i < len(arg); i++ {
		switch arg[i] {
		case '\\':
			backslashes++
		case '"':
			for j := 0; j < backslashes*2+1; j++ {
				b.WriteByte('\\')
			}
			b.WriteByte('"')
			backslashes = 0
		default:
			for j := 0; j < backslashes; j++ {
				b.WriteByte('\\')
			}
			backslashes = 0
			b.WriteByte(arg[i])
		}
	}
	for j := 0; j < backslashes*2; j++ {
		b.WriteByte('\\')
	}
	b.WriteByte('"')
	return b.String()
}
