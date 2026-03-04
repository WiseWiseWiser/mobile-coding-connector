package lib

import (
	"fmt"
	"os"

	"github.com/xhd2015/lifelog-private/ai-critic/server/env"
)

const (
	EnvQuickTestDefaultConfig = "QUICK_TEST_DEFAULT_CONFIG"

	QuickTestDefaultConfigLocal = "local"
	QuickTestDefaultConfigHome  = "home"
	QuickTestDefaultConfigUnset = ""
)

// QuickTestBaseURL returns the base URL for the quick-test server,
// constructed from QUICK_TEST_DOMAIN and QUICK_TEST_PORT env vars.
// Defaults to http://localhost:<defaultPort>.
// If QUICK_TEST_PORT is "UNSET", the port is omitted (e.g. https://example.com).
func QuickTestBaseURL(defaultPort int) string {
	domain := os.Getenv(env.EnvQuickTestDomain)
	if domain == "" {
		domain = "localhost"
	}

	portStr := os.Getenv(env.EnvQuickTestPort)
	if portStr == env.QuickTestPortUnset {
		scheme := "https"
		if domain == "localhost" || domain == "127.0.0.1" {
			scheme = "http"
		}
		return fmt.Sprintf("%s://%s", scheme, domain)
	}

	port := defaultPort
	if portStr != "" {
		fmt.Sscanf(portStr, "%d", &port)
	}

	scheme := "http"
	if domain != "localhost" && domain != "127.0.0.1" {
		scheme = "https"
	}
	return fmt.Sprintf("%s://%s:%d", scheme, domain, port)
}
