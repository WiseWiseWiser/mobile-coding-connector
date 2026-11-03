package main

import (
	"strings"
	"testing"
)

func TestInvalidCookieNameRejectedBeforeBrowserOrCredentials(t *testing.T) {
	for _, name := range []string{"", "bad name", "svc-auth-token", "__Host-token"} {
		if err := Handle([]string{"--auth-cookie-name", name}); err == nil || !strings.Contains(err.Error(), "invalid --auth-cookie-name") {
			t.Fatalf("invalid name %q was not rejected before loading credentials or starting Chrome: %v", name, err)
		}
	}
}
