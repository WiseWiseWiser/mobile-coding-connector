package grokusage

import (
	"github.com/xhd2015/agent-pro/agent/grok/tty"
)

// UsageInfo holds parsed grok show-usage output fields.
type UsageInfo struct {
	WeeklyLimit string // e.g. "6%"
	NextReset   string // e.g. "July 9, 16:55 PT"
}

// ParseShowUsageOutput extracts Weekly limit and Next reset lines from command stdout.
func ParseShowUsageOutput(stdout string) (*UsageInfo, error) {
	info, err := tty.ParseShowUsageOutput(stdout)
	if err != nil {
		return nil, err
	}
	return &UsageInfo{
		WeeklyLimit: info.WeeklyLimit,
		NextReset:   info.NextReset,
	}, nil
}