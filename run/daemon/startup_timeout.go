package daemon

import (
	"fmt"
	"os"
	"time"
)

const keepAliveStartupTimeoutEnv = "AI_CRITIC_KEEPALIVE_STARTUP_TIMEOUT"

// ResolveStartupTimeout returns the configured startup timeout from CLI flag or env.
// Default is 60s; minimum is 10s.
func ResolveStartupTimeout(flagValue string) (time.Duration, error) {
	raw := flagValue
	if raw == "" {
		raw = os.Getenv(keepAliveStartupTimeoutEnv)
	}
	if raw == "" {
		return DefaultStartupTimeout, nil
	}

	d, err := time.ParseDuration(raw)
	if err != nil {
		return 0, fmt.Errorf("invalid startup timeout %q: %w", raw, err)
	}
	if d < MinStartupTimeout {
		return 0, fmt.Errorf("startup timeout %v is below minimum %v", d, MinStartupTimeout)
	}
	return d, nil
}

// StartupBackoffDelay returns exponential backoff before the next spawn attempt
// after consecutive startup-timeout failures: 3s, 6s, 12s, … capped at 60s.
func StartupBackoffDelay(consecutiveFailures int) time.Duration {
	if consecutiveFailures <= 0 {
		return StartupBackoffBase
	}
	delay := StartupBackoffBase
	for i := 1; i < consecutiveFailures; i++ {
		if delay >= StartupBackoffMax {
			return StartupBackoffMax
		}
		delay *= 2
	}
	if delay > StartupBackoffMax {
		return StartupBackoffMax
	}
	return delay
}