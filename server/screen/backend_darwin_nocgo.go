//go:build darwin && !cgo

package screen

import (
	"fmt"
	"os/exec"
)

// newBackend returns a view-only backend: input injection needs CoreGraphics,
// which requires cgo. Shipping without cgo still gives a usable screen mirror.
func newBackend() backend {
	return &unavailableBackend{
		name:   "none",
		reason: "this build has no cgo, so input injection is unavailable (rebuild with CGO_ENABLED=1 for full control)",
	}
}

func promptPermission(which string) error {
	switch which {
	case "accessibility", "input", "":
		return exec.Command("open", "x-apple.systempreferences:com.apple.preference.security?Privacy_Accessibility").Run()
	case "screen", "screen-recording":
		return exec.Command("open", "x-apple.systempreferences:com.apple.preference.security?Privacy_ScreenCapture").Run()
	}
	return fmt.Errorf("unknown permission %q", which)
}
