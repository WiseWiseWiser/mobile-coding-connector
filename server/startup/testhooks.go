package startup

import (
	"os"
	"strconv"
	"time"
)

const (
	envCoreDelayMs      = "AI_CRITIC_TEST_CORE_DELAY_MS"
	envExtensionDelayMs = "AI_CRITIC_TEST_EXTENSION_DELAY_MS"
	envSkipExtension    = "AI_CRITIC_TEST_SKIP_EXTENSION"
)

// CoreStartupDelay returns injected delay from AI_CRITIC_TEST_CORE_DELAY_MS.
func CoreStartupDelay() time.Duration {
	raw := os.Getenv(envCoreDelayMs)
	if raw == "" {
		return 0
	}
	ms, err := strconv.Atoi(raw)
	if err != nil || ms <= 0 {
		return 0
	}
	return time.Duration(ms) * time.Millisecond
}

// ExtensionStartupDelay returns injected delay from AI_CRITIC_TEST_EXTENSION_DELAY_MS.
func ExtensionStartupDelay() time.Duration {
	raw := os.Getenv(envExtensionDelayMs)
	if raw == "" {
		return 0
	}
	ms, err := strconv.Atoi(raw)
	if err != nil || ms <= 0 {
		return 0
	}
	return time.Duration(ms) * time.Millisecond
}

// SkipExtensionStartup reports whether AI_CRITIC_TEST_SKIP_EXTENSION=1.
func SkipExtensionStartup() bool {
	return os.Getenv(envSkipExtension) == "1"
}