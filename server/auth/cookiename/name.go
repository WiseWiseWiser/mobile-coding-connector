// Package cookiename defines the management Cookie wire contract without importing
// server runtime state. Standalone browser/HTTP tooling can share the same rules.
package cookiename

import (
	"fmt"
	"net/http"
	"strings"
)

const (
	DefaultName = "ai-critic-token"
	ServiceName = "svc-auth-token"
)

// Validate checks a management Cookie name. This application does not issue
// Secure cookies, so browser prefixes requiring Secure attributes are unsupported.
func Validate(name string) error {
	if err := (&http.Cookie{Name: name, Value: "validation"}).Valid(); err != nil {
		return fmt.Errorf("invalid HTTP cookie name %q", name)
	}
	if name == ServiceName {
		return fmt.Errorf("cookie name %q is reserved for service authentication", name)
	}
	lower := strings.ToLower(name)
	for _, prefix := range []string{"__host-", "__secure-", "__http-"} {
		if strings.HasPrefix(lower, prefix) {
			return fmt.Errorf("cookie name %q requires Secure cookie attributes not supported by this server", name)
		}
	}
	return nil
}
