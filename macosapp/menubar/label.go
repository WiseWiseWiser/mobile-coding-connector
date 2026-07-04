package menubar

import "unicode/utf8"

const maxLabelLen = 40

// FormatGrokLabel maps daemon grok usage fields to a compact menu-bar label.
func FormatGrokLabel(status, weeklyLimit, errorMsg string) string {
	const prefix = "Grok "
	switch status {
	case "ready":
		return prefix + weeklyLimit
	case "loading":
		return prefix + "..."
	case "error":
		return truncateRunes(prefix+errorMsg, maxLabelLen)
	default:
		return prefix + "..."
	}
}

// TestExported_MaxLabelLen returns the maximum menu-bar label length in runes.
func TestExported_MaxLabelLen() int {
	return maxLabelLen
}

func truncateRunes(s string, max int) string {
	if utf8.RuneCountInString(s) <= max {
		return s
	}
	runes := []rune(s)
	ellipsis := "…"
	ellipsisLen := utf8.RuneCountInString(ellipsis)
	keep := max - ellipsisLen
	if keep < 0 {
		keep = 0
	}
	if keep > len(runes) {
		keep = len(runes)
	}
	return string(runes[:keep]) + ellipsis
}