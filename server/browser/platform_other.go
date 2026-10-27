//go:build !darwin && !linux

package browser

import (
	"context"
	"fmt"
)

func findBinary() string { return "" }

func (m *Manager) spawnDaemon() error {
	return fmt.Errorf("browser control is only supported on macOS and Linux")
}

func openInChrome(_ context.Context, _ string) error {
	return fmt.Errorf("browser control is only supported on macOS and Linux")
}

func openNewChromeWindow(_ context.Context, _ string) error {
	return fmt.Errorf("browser control is only supported on macOS and Linux")
}
