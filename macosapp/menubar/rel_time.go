package menubar

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"time"
)

var (
	grokResetRE  = regexp.MustCompile(`^(\w+)\s+(\d+),\s+(\d+):(\d+)(?::(\d+))?\s+PT$`)
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

func parseResetTime(reset string, now time.Time) (time.Time, bool) {
	reset = strings.TrimSpace(reset)
	if reset == "" {
		return time.Time{}, false
	}

	if matches := grokResetRE.FindStringSubmatch(reset); matches != nil {
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
		if matches[5] != "" {
			second, err = strconv.Atoi(matches[5])
			if err != nil {
				return time.Time{}, false
			}
		}

		year := now.In(ptLocation).Year()
		resetTime := time.Date(year, month, day, hour, minute, second, 0, ptLocation)
		if resetTime.Before(now) {
			resetTime = resetTime.AddDate(1, 0, 0)
		}
		return resetTime, true
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

// FormatTimeLeft parses provider reset strings and returns compact relative countdown text.
func FormatTimeLeft(reset string, now time.Time) string {
	resetTime, ok := parseResetTime(reset, now)
	if !ok {
		return ""
	}

	remaining := resetTime.Sub(now)
	if remaining <= 0 {
		return "left 0min"
	}

	hours := remaining.Hours()
	if hours >= 24 {
		days := int(remaining / (24 * time.Hour))
		return fmt.Sprintf("left %dd", days)
	}
	if hours >= 1 {
		hrs := int(remaining / time.Hour)
		return fmt.Sprintf("left %dh", hrs)
	}

	mins := int(remaining / time.Minute)
	if mins < 1 {
		mins = 1
	}
	return fmt.Sprintf("left %dmin", mins)
}

// FormatResetSuffix returns a comma-prefixed relative suffix for dropdown parentheses.
func FormatResetSuffix(reset string, now time.Time) string {
	left := FormatTimeLeft(reset, now)
	if left == "" {
		return ""
	}
	return ", " + left
}