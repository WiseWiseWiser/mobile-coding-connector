package menubar

import (
	"fmt"
	"strings"
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
// Parses raw provider reset strings (legacy path used by low-level doctests).
func FormatGrokDropdownLine(status, weeklyLimit, reset, errorMsg string, now time.Time) string {
	switch status {
	case "ready":
		display := FormatResetDisplay(reset, now)
		return ComposeGrokDropdownLine(status, weeklyLimit, "Weekly", display, FormatTimeLeft(reset, now), errorMsg)
	case "loading":
		return ComposeGrokDropdownLine(status, weeklyLimit, "Weekly", "", "", errorMsg)
	case "error":
		return ComposeGrokDropdownLine(status, weeklyLimit, "Weekly", "", "", errorMsg)
	default:
		return ComposeGrokDropdownLine(status, weeklyLimit, "Weekly", "", "", errorMsg)
	}
}

// FormatCodexDropdownLine formats a single-line codex usage row for the menu dropdown.
// Parses raw provider reset strings (legacy path used by low-level doctests).
func FormatCodexDropdownLine(status, monthlyUsage, creditsUsed, creditsTotal, reset, errorMsg string, now time.Time) string {
	switch status {
	case "ready":
		display := FormatResetDisplay(reset, now)
		return ComposeCodexDropdownLine(status, monthlyUsage, creditsUsed, creditsTotal, display, FormatTimeLeft(reset, now), errorMsg)
	case "loading":
		return ComposeCodexDropdownLine(status, monthlyUsage, creditsUsed, creditsTotal, "", "", errorMsg)
	case "error":
		return ComposeCodexDropdownLine(status, monthlyUsage, creditsUsed, creditsTotal, "", "", errorMsg)
	default:
		return ComposeCodexDropdownLine(status, monthlyUsage, creditsUsed, creditsTotal, "", "", errorMsg)
	}
}

// ComposeGrokBody builds the grok dropdown text without the provider prefix,
// so a usage item can render it under its own label.
// periodLabel empty defaults to "Weekly" (historical menu wording).
func ComposeGrokBody(status, weeklyLimit, periodLabel, resetDisplay, timeLeft, errorMsg string) string {
	if strings.TrimSpace(periodLabel) == "" {
		periodLabel = "Weekly"
	}
	switch status {
	case "ready":
		body := fmt.Sprintf("%s(%s), Reset %s", weeklyLimit, periodLabel, resetDisplay)
		if timeLeft != "" {
			body += ", " + timeLeft
		}
		return body
	case "error":
		return fmt.Sprintf("Error: %s", errorMsg)
	default:
		return "Loading..."
	}
}

// ComposeGrokDropdownLine builds the dropdown line from structured API fields only.
// Does not parse next_reset; time_left is appended only when non-empty.
// periodLabel empty defaults to "Weekly" (historical menu wording).
func ComposeGrokDropdownLine(status, weeklyLimit, periodLabel, resetDisplay, timeLeft, errorMsg string) string {
	return "Grok: " + ComposeGrokBody(status, weeklyLimit, periodLabel, resetDisplay, timeLeft, errorMsg)
}

// PeriodLabelForAPI maps API period tokens to menu display labels.
func PeriodLabelForAPI(period string) string {
	switch strings.ToLower(strings.TrimSpace(period)) {
	case "monthly":
		return "Monthly"
	case "weekly":
		return "Weekly"
	default:
		return "Weekly"
	}
}

// ComposeCodexBody builds the codex dropdown text without the provider prefix,
// so a usage item can render it under its own label.
func ComposeCodexBody(status, monthlyUsage, creditsUsed, creditsTotal, resetDisplay, timeLeft, errorMsg string) string {
	switch status {
	case "ready":
		body := fmt.Sprintf("%s(Monthly) %s/%s, Reset %s", monthlyUsage, creditsUsed, creditsTotal, resetDisplay)
		if timeLeft != "" {
			body += ", " + timeLeft
		}
		return body
	case "error":
		return fmt.Sprintf("Error: %s", errorMsg)
	default:
		return "Loading..."
	}
}

// ComposeCodexDropdownLine builds the dropdown line from structured API fields only.
// Does not parse next_reset; time_left is appended only when non-empty.
func ComposeCodexDropdownLine(status, monthlyUsage, creditsUsed, creditsTotal, resetDisplay, timeLeft, errorMsg string) string {
	return "Codex: " + ComposeCodexBody(status, monthlyUsage, creditsUsed, creditsTotal, resetDisplay, timeLeft, errorMsg)
}

// TestExported_MaxLabelLen returns the maximum menu-bar label length in runes.
func TestExported_MaxLabelLen() int {
	return maxLabelLen
}

// TruncateRunes shortens s to max runes, appending an ellipsis when cut.
func TruncateRunes(s string, max int) string {
	return truncateRunes(s, max)
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