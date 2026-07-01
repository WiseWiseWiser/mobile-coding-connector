package terminal

import "strings"

// ShellQuote returns a POSIX shell single-quoted representation of s.
func ShellQuote(s string) string {
	if s == "" {
		return "''"
	}
	return "'" + strings.ReplaceAll(s, "'", "'\"'\"'") + "'"
}
