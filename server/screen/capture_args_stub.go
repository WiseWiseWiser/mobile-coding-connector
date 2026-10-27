//go:build !darwin && !linux

package screen

import (
	"fmt"
	"runtime"
	"syscall"
)

func captureArgs(maxWidth, fps int) ([]string, error) {
	return nil, fmt.Errorf("screen mirroring is not supported on %s yet (macOS and X11 Linux only)", runtime.GOOS)
}

func sysProcAttr() *syscall.SysProcAttr { return nil }

func platformName() string { return runtime.GOOS }

// captureToolName names the capture path actually used on this platform.
func captureToolName() string { return "" }
