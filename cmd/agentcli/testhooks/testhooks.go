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
)

// ApplyFromEnv reads test-only environment variables. Call at process startup.
func ApplyFromEnv() {
	defaultPortOverride = 0
	reachabilityMode = ""
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