package run

import (
	"fmt"
	"io"

	"github.com/xhd2015/ai-critic/server/auth"
)

// A pointer keeps an omitted option distinct from an explicitly empty name.
func resolveAuthCookieName(value *string) (string, error) {
	name := auth.DefaultCookieName
	if value != nil {
		name = *value
	}
	if err := auth.ValidateCookieName(name); err != nil {
		return "", fmt.Errorf("invalid --auth-cookie-name: %w", err)
	}
	return name, nil
}

func reportAuthCookieName(out, errOut io.Writer, name string) {
	fmt.Fprintf(out, "auth cookie name: %s\n", name)
	if name != auth.DefaultCookieName {
		fmt.Fprintf(errOut, "warning: sessions using %s are not reused; sign in again on this instance\n", auth.DefaultCookieName)
	}
}
