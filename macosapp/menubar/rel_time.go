package menubar

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"time"
)

var (
	// Explicit PT (legacy Grok); wall clock is America/Los_Angeles.
	grokResetPTRE = regexp.MustCompile(`^(\w+)\s+(\d+),\s+(\d+):(\d+)(?::(\d+))?\s+PT$`)
	// Explicit UTC.
	grokResetUTCRE = regexp.MustCompile(`(?i)^(\w+)\s+(\d+),\s+(\d+):(\d+)(?::(\d+))?\s+UTC$`)
	// No timezone: bare wall clock interpreted as machine local time.
	grokResetLocalRE = regexp.MustCompile(`^(\w+)\s+(\d+),\s+(\d+):(\d+)(?::(\d+))?$`)
	// Back-compat name used by FormatResetDisplay classification.
	grokResetRE  = grokResetPTRE
	codexResetRE = regexp.MustCompile(`^(\d+):(\d+)\s+on\s+(\d+)\s+(\w+)$`)
)

var monthByName = map[string]time.Month{
	"january":   time.January,
	"jan":       time.January,
	"february":  time.February,
	"feb":       time.February,
	"march":     time.March,
	"mar":       time.March,
	"april":     time.April,
	"apr":       time.April,
	"may":       time.May,
	"june":      time.June,
	"jun":       time.June,
	"july":      time.July,
	"jul":       time.July,
	"august":    time.August,
	"aug":       time.August,
	"september": time.September,
	"sep":       time.September,
	"sept":      time.September,
	"october":   time.October,
	"oct":       time.October,
	"november":  time.November,
	"nov":       time.November,
	"december":  time.December,
	"dec":       time.December,
}

var ptLocation *time.Location

func init() {
	loc, err := time.LoadLocation("America/Los_Angeles")
	if err != nil {
		ptLocation = time.FixedZone("PT", -8*3600)
		return
	}
	ptLocation = loc
}

func parseMonth(name string) (time.Month, bool) {
	month, ok := monthByName[strings.ToLower(name)]
	return month, ok
}

func parseGrokWallClock(matches []string, now time.Time, loc *time.Location) (time.Time, bool) {
	month, ok := parseMonth(matches[1])
	if !ok {
		return time.Time{}, false
	}
	day, err := strconv.Atoi(matches[2])
	if err != nil {
		return time.Time{}, false
	}
	hour, err := strconv.Atoi(matches[3])
	if err != nil {
		return time.Time{}, false
	}
	minute, err := strconv.Atoi(matches[4])
	if err != nil {
		return time.Time{}, false
	}
	second := 0
	if len(matches) > 5 && matches[5] != "" {
		second, err = strconv.Atoi(matches[5])
		if err != nil {
			return time.Time{}, false
		}
	}

	year := now.In(loc).Year()
	resetTime := time.Date(year, month, day, hour, minute, second, 0, loc)
	if resetTime.Before(now) {
		resetTime = resetTime.AddDate(1, 0, 0)
	}
	return resetTime, true
}

func parseResetTime(reset string, now time.Time) (time.Time, bool) {
	reset = strings.TrimSpace(reset)
	if reset == "" {
		return time.Time{}, false
	}

	// Ordered: explicit PT, explicit UTC, then bare local wall clock.
	if matches := grokResetPTRE.FindStringSubmatch(reset); matches != nil {
		return parseGrokWallClock(matches, now, ptLocation)
	}
	if matches := grokResetUTCRE.FindStringSubmatch(reset); matches != nil {
		return parseGrokWallClock(matches, now, time.UTC)
	}
	if matches := grokResetLocalRE.FindStringSubmatch(reset); matches != nil {
		return parseGrokWallClock(matches, now, now.Location())
	}

	if matches := codexResetRE.FindStringSubmatch(reset); matches != nil {
		hour, err := strconv.Atoi(matches[1])
		if err != nil {
			return time.Time{}, false
		}
		minute, err := strconv.Atoi(matches[2])
		if err != nil {
			return time.Time{}, false
		}
		day, err := strconv.Atoi(matches[3])
		if err != nil {
			return time.Time{}, false
		}
		month, ok := parseMonth(matches[4])
		if !ok {
			return time.Time{}, false
		}

		loc := now.Location()
		year := now.In(loc).Year()
		resetTime := time.Date(year, month, day, hour, minute, 0, 0, loc)
		if resetTime.Before(now) {
			resetTime = resetTime.AddDate(1, 0, 0)
		}
		return resetTime, true
	}

	return time.Time{}, false
}

// FormatResetDisplay parses provider reset strings and returns local wall-clock display text.
func FormatResetDisplay(reset string, now time.Time) string {
	reset = strings.TrimSpace(reset)
	if reset == "" {
		return reset
	}

	isGrok := grokResetPTRE.MatchString(reset) ||
		grokResetUTCRE.MatchString(reset) ||
		grokResetLocalRE.MatchString(reset)
	isCodex := codexResetRE.MatchString(reset)

	resetTime, ok := parseResetTime(reset, now)
	if !ok {
		return reset
	}

	local := resetTime.In(now.Location())
	if isGrok {
		return fmt.Sprintf("%s %d, %02d:%02d", local.Month().String(), local.Day(), local.Hour(), local.Minute())
	}
	if isCodex {
		return local.Format("Jan 2, 15:04")
	}
	return reset
}

// FormatTimeLeft parses provider reset strings and returns compact relative countdown text.
func FormatTimeLeft(reset string, now time.Time) string {
	resetTime, ok := parseResetTime(reset, now)
	if !ok {
		return ""
	}
	return FormatTimeLeftFromInstant(resetTime, now)
}

// FormatTimeLeftFromInstant formats a countdown from an absolute reset instant.
// Unit policy matches FormatTimeLeft: left Nd, left NdNh, left NhNm, left Nm, left 0m.
func FormatTimeLeftFromInstant(resetAt, now time.Time) string {
	if resetAt.IsZero() {
		return ""
	}
	remaining := resetAt.Sub(now)
	if remaining <= 0 {
		return "left 0m"
	}

	totalHours := int(remaining / time.Hour)
	if totalHours >= 24 {
		days := int(remaining / (24 * time.Hour))
		hours := totalHours % 24
		if hours == 0 {
			return fmt.Sprintf("left %dd", days)
		}
		return fmt.Sprintf("left %dd%dh", days, hours)
	}
	if totalHours >= 1 {
		minutes := int(remaining/time.Minute) % 60
		if minutes == 0 {
			return fmt.Sprintf("left %dh", totalHours)
		}
		return fmt.Sprintf("left %dh%dm", totalHours, minutes)
	}

	mins := int(remaining / time.Minute)
	if mins < 1 {
		mins = 1
	}
	return fmt.Sprintf("left %dm", mins)
}

// ResolveStructuredReset derives A (reset_at) + B (reset_display, time_left) from a
// raw provider next_reset string. Unparseable reset yields empty reset_at and time_left;
// reset_display falls back to FormatResetDisplay (raw when unparseable).
func ResolveStructuredReset(nextReset string, now time.Time) (resetAt, resetDisplay, timeLeft string) {
	nextReset = strings.TrimSpace(nextReset)
	if nextReset == "" {
		return "", "", ""
	}
	resetDisplay = FormatResetDisplay(nextReset, now)
	if resetTime, ok := parseResetTime(nextReset, now); ok {
		resetAt = resetTime.Format(time.RFC3339)
		timeLeft = FormatTimeLeftFromInstant(resetTime, now)
	}
	return resetAt, resetDisplay, timeLeft
}

// FormatResetSuffix returns a comma-prefixed relative suffix for dropdown parentheses.
func FormatResetSuffix(reset string, now time.Time) string {
	left := FormatTimeLeft(reset, now)
	if left == "" {
		return ""
	}
	return ", " + left
}