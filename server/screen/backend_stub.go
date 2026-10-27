//go:build !darwin && !linux

package screen

import "runtime"

func newBackend() backend {
	return &unavailableBackend{
		name:   "none",
		reason: "screen control is not supported on " + runtime.GOOS + " yet (macOS and X11 Linux only)",
	}
}

func promptPermission(string) error { return errNotSupported }
