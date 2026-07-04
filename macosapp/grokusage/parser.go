package grokusage

import (
	"fmt"
	"strings"
)

// UsageInfo holds parsed grok show-usage output fields.
type UsageInfo struct {
	WeeklyLimit string // e.g. "6%"
	NextReset   string // e.g. "July 9, 16:55 PT"
}

// ParseShowUsageOutput extracts Weekly limit and Next reset lines from command stdout.
func ParseShowUsageOutput(stdout string) (*UsageInfo, error) {
	var weekly, reset string
	for _, line := range strings.Split(stdout, "\n") {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, "Weekly limit:") {
			weekly = strings.TrimSpace(strings.TrimPrefix(line, "Weekly limit:"))
		}
		if strings.HasPrefix(line, "Next reset:") {
			reset = strings.TrimSpace(strings.TrimPrefix(line, "Next reset:"))
		}
	}
	if weekly == "" {
		return nil, fmt.Errorf("missing weekly limit")
	}
	if reset == "" {
		return nil, fmt.Errorf("missing next reset")
	}
	return &UsageInfo{WeeklyLimit: weekly, NextReset: reset}, nil
}