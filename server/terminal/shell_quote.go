package terminal

import "strings"

// ShellQuote wraps a string in single quotes for safe shell usage.
func ShellQuote(s string) string {
	// If it's simple enough, no quoting needed
	safe := true
	for _, c := range s {
		if !((c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z') || (c >= '0' && c <= '9') || c == '/' || c == '.' || c == '-' || c == '_') {
			safe = false
			break
		}
	}
	if safe && s != "" {
		return s
	}
	// Escape single quotes: replace ' with '\''
	return "'" + strings.ReplaceAll(s, "'", `'\''`) + "'"
}
