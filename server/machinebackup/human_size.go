package machinebackup

import (
	"fmt"
	"strconv"
	"strings"
)

const defaultLargeDirThresholdBytes = 10 * 1024 * 1024

// ParseHumanSize parses human-readable sizes such as 40MB, 50M, 1G, 1GB (binary units).
func ParseHumanSize(s string) (int64, error) {
	s = strings.TrimSpace(s)
	if s == "" {
		return 0, fmt.Errorf("invalid size: empty string")
	}

	upper := strings.ToUpper(s)
	if upper == "0" || upper == "0B" {
		return 0, nil
	}

	units := []struct {
		suffix string
		mult   int64
	}{
		{"TB", 1024 * 1024 * 1024 * 1024},
		{"T", 1024 * 1024 * 1024 * 1024},
		{"GB", 1024 * 1024 * 1024},
		{"G", 1024 * 1024 * 1024},
		{"MB", 1024 * 1024},
		{"M", 1024 * 1024},
		{"KB", 1024},
		{"K", 1024},
		{"B", 1},
	}

	for _, u := range units {
		if strings.HasSuffix(upper, u.suffix) {
			numStr := strings.TrimSpace(s[:len(s)-len(u.suffix)])
			if numStr == "" {
				return 0, fmt.Errorf("invalid size: %q", s)
			}
			val, err := strconv.ParseFloat(numStr, 64)
			if err != nil {
				return 0, fmt.Errorf("invalid size: %q", s)
			}
			if val < 0 {
				return 0, fmt.Errorf("invalid size: %q", s)
			}
			return int64(val * float64(u.mult)), nil
		}
	}

	return 0, fmt.Errorf("invalid size: %q", s)
}

// EffectiveLargeDirThreshold returns opts threshold or the 10 MB default.
func EffectiveLargeDirThreshold(bytes int64) int64 {
	if bytes > 0 {
		return bytes
	}
	return defaultLargeDirThresholdBytes
}

// ResolveLargeDirThresholdBytes returns CLI threshold when set, else persisted user config, else 10 MB.
func ResolveLargeDirThresholdBytes(home string, cliBytes int64) (int64, error) {
	if cliBytes > 0 {
		return cliBytes, nil
	}
	user, err := LoadUserBackupConfig(home)
	if err != nil {
		return 0, err
	}
	if user != nil {
		if threshold := strings.TrimSpace(user.LargeDirThreshold); threshold != "" {
			return ParseHumanSize(threshold)
		}
	}
	return defaultLargeDirThresholdBytes, nil
}