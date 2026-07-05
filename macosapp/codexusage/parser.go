package codexusage

import (
	"fmt"
	"strconv"
	"strings"
)

// UsageInfo holds parsed codex show-status output fields.
type UsageInfo struct {
	MonthlyUsage string // e.g. "58%"
	CreditsUsed  string // e.g. "6,519"
	CreditsTotal string // e.g. "11,250"
	NextReset    string // e.g. "08:00 on 1 Aug"
}

// ParseStatusOutput extracts Monthly usage, Credits used, and Next reset lines from command stdout.
func ParseStatusOutput(stdout string) (*UsageInfo, error) {
	var monthly, creditsUsed, creditsTotal, reset string
	for _, line := range strings.Split(stdout, "\n") {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, "Monthly usage:") {
			monthly = strings.TrimSpace(strings.TrimPrefix(line, "Monthly usage:"))
		}
		if strings.HasPrefix(line, "Credits used:") {
			rest := strings.TrimSpace(strings.TrimPrefix(line, "Credits used:"))
			parts := strings.Split(rest, " of ")
			if len(parts) == 2 {
				if used, err := strconv.Atoi(strings.TrimSpace(parts[0])); err == nil {
					creditsUsed = formatWithCommas(used)
				}
				if total, err := strconv.Atoi(strings.TrimSpace(parts[1])); err == nil {
					creditsTotal = formatWithCommas(total)
				}
			}
		}
		if strings.HasPrefix(line, "Next reset:") {
			reset = strings.TrimSpace(strings.TrimPrefix(line, "Next reset:"))
		}
	}
	if monthly == "" {
		return nil, fmt.Errorf("missing monthly usage")
	}
	if creditsUsed == "" || creditsTotal == "" {
		return nil, fmt.Errorf("missing credits used")
	}
	if reset == "" {
		return nil, fmt.Errorf("missing next reset")
	}
	return &UsageInfo{
		MonthlyUsage: monthly,
		CreditsUsed:  creditsUsed,
		CreditsTotal: creditsTotal,
		NextReset:    reset,
	}, nil
}

func formatWithCommas(n int) string {
	if n < 0 {
		return strconv.Itoa(n)
	}
	s := strconv.Itoa(n)
	if len(s) <= 3 {
		return s
	}
	var parts []string
	for len(s) > 3 {
		parts = append([]string{s[len(s)-3:]}, parts...)
		s = s[:len(s)-3]
	}
	if s != "" {
		parts = append([]string{s}, parts...)
	}
	return strings.Join(parts, ",")
}