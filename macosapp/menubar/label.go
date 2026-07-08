package menubar

import (
	"fmt"
	"time"
	"unicode/utf8"
)

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
		return prefix + "err"
	default:
		return prefix + "..."
	}
}

func formatCodexLabel(status, monthlyUsage, errorMsg string) string {
	const prefix = "Codex "
	switch status {
	case "ready":
		return prefix + monthlyUsage
	case "loading":
		return prefix + "..."
	case "error":
		return prefix + "err"
	default:
		return prefix + "..."
	}
}

// FormatMenuBarLabel selects the menu-bar title from display mode and provider status fields.
func FormatMenuBarLabel(
	mode string,
	rotatingIndex int,
	grokStatus, grokWeekly, grokError string,
	codexStatus, codexMonthly, codexError string,
) string {
	switch mode {
	case "grok":
		return FormatGrokLabel(grokStatus, grokWeekly, grokError)
	case "codex":
		return formatCodexLabel(codexStatus, codexMonthly, codexError)
	case "rotating":
		if rotatingIndex%2 == 1 {
			return formatCodexLabel(codexStatus, codexMonthly, codexError)
		}
		return FormatGrokLabel(grokStatus, grokWeekly, grokError)
	default:
		return FormatGrokLabel(grokStatus, grokWeekly, grokError)
	}
}

// FormatGrokDropdownLine formats a single-line grok usage row for the menu dropdown.
func FormatGrokDropdownLine(status, weeklyLimit, reset, errorMsg string, now time.Time) string {
	switch status {
	case "ready":
		display := FormatResetDisplay(reset, now)
		line := fmt.Sprintf("Grok: %s(Weekly), Reset %s", weeklyLimit, display)
		if timeLeft := FormatTimeLeft(reset, now); timeLeft != "" {
			line += ", " + timeLeft
		}
		return line
	case "loading":
		return "Grok: Loading..."
	case "error":
		return fmt.Sprintf("Grok: Error: %s", errorMsg)
	default:
		return "Grok: Loading..."
	}
}

// FormatCodexDropdownLine formats a single-line codex usage row for the menu dropdown.
func FormatCodexDropdownLine(status, monthlyUsage, creditsUsed, creditsTotal, reset, errorMsg string, now time.Time) string {
	switch status {
	case "ready":
		display := FormatResetDisplay(reset, now)
		line := fmt.Sprintf("Codex: %s(Monthly) %s/%s, Reset %s", monthlyUsage, creditsUsed, creditsTotal, display)
		if timeLeft := FormatTimeLeft(reset, now); timeLeft != "" {
			line += ", " + timeLeft
		}
		return line
	case "loading":
		return "Codex: Loading..."
	case "error":
		return fmt.Sprintf("Codex: Error: %s", errorMsg)
	default:
		return "Codex: Loading..."
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