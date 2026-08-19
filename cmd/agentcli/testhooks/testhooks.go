package testhooks

import (
	"os"
	"strconv"
	"strings"
)

const (
	envDefaultPort  = "AGENTCLI_TEST_DEFAULT_PORT"
	envReachability = "AGENTCLI_TEST_REACHABILITY"
)

var (
	defaultPortOverride int
	reachabilityMode    string // "", "up", "down"
	homeOverride        string
)

// ApplyFromEnv reads test-only environment variables. Call at process startup.
func ApplyFromEnv() {
	defaultPortOverride = 0
	reachabilityMode = ""
	homeOverride = ""
	if v := strings.TrimSpace(os.Getenv(envDefaultPort)); v != "" {
		if p, err := strconv.Atoi(v); err == nil && p > 0 {
			defaultPortOverride = p
		}
	}
	reachabilityMode = strings.ToLower(strings.TrimSpace(os.Getenv(envReachability)))
}

// AppendDefaultPortEnv sets AGENTCLI_TEST_DEFAULT_PORT for the child process.
func AppendDefaultPortEnv(env []string, port int) []string {
	return append(env, envDefaultPort+"="+strconv.Itoa(port))
}

// AppendReachabilityEnv sets AGENTCLI_TEST_REACHABILITY=up|down for the child.
func AppendReachabilityEnv(env []string, up bool) []string {
	val := "down"
	if up {
		val = "up"
	}
	return append(env, envReachability+"="+val)
}

// EffectiveDefaultPort returns the built-in default port, overridden by test env when set.
func EffectiveDefaultPort(builtin int) int {
	if defaultPortOverride > 0 {
		return defaultPortOverride
	}
	return builtin
}

// ReachabilityForced returns (forced, up). forced is true when env mocks reachability.
func ReachabilityForced() (forced bool, up bool) {
	switch reachabilityMode {
	case "up":
		return true, true
	case "down":
		return true, false
	default:
		return false, false
	}
}

// SetHomeOverride scopes agentcli config and worktree roots to dir for in-process
// tests. Empty clears. Must be paired with agentcli.Run under a process mutex;
// does not call os.Setenv.
func SetHomeOverride(dir string) {
	homeOverride = strings.TrimSpace(dir)
}

// SetDefaultPortForTest sets the in-process default port override (0 clears).
func SetDefaultPortForTest(port int) {
	if port < 0 {
		port = 0
	}
	defaultPortOverride = port
}

// SetReachabilityForTest sets in-process reachability mock: "", "up", or "down".
func SetReachabilityForTest(mode string) {
	reachabilityMode = strings.ToLower(strings.TrimSpace(mode))
}

// ResetInProcessOverrides clears home/port/reachability overrides after an
// in-process agentcli.Run. Safe to call from defer under the suite mutex.
// Also clears terminals list/focus inject hooks and their counters.
func ResetInProcessOverrides() {
	homeOverride = ""
	defaultPortOverride = 0
	reachabilityMode = ""
	resetTerminalsHooks()
}

// UserHomeDir returns the home override when set, otherwise os.UserHomeDir.
// Used by agentcli for config paths and project worktree roots without Setenv.
func UserHomeDir() (string, error) {
	if homeOverride != "" {
		return homeOverride, nil
	}
	return os.UserHomeDir()
}
