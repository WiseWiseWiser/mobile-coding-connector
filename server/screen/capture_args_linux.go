//go:build linux

package screen

import (
	"fmt"
	"os"
	"strconv"
	"syscall"
)

// captureArgs builds the ffmpeg invocation for X11. -draw_mouse 0 is required:
// x11grab draws the pointer by default, which would duplicate the browser-drawn
// cursor and leave a trail behind it.
func captureArgs(maxWidth, fps int) ([]string, error) {
	display := os.Getenv("DISPLAY")
	if display == "" {
		if os.Getenv("WAYLAND_DISPLAY") != "" || os.Getenv("XDG_SESSION_TYPE") == "wayland" {
			return nil, fmt.Errorf("wayland is not supported yet: screen mirroring needs an X11 session (log in with Xorg, or use a VNC server)")
		}
		return nil, fmt.Errorf("DISPLAY is not set: no X11 display to capture")
	}
	return []string{
		"-hide_banner", "-loglevel", "error",
		"-fflags", "nobuffer",
		"-f", "x11grab",
		"-draw_mouse", "0",
		"-framerate", strconv.Itoa(fps),
		"-i", display,
		"-an",
		"-vf", fmt.Sprintf("scale=%d:-2:flags=bilinear", maxWidth),
		"-q:v", "3",
		"-f", "mjpeg",
		"-flush_packets", "1",
		"pipe:1",
	}, nil
}

func sysProcAttr() *syscall.SysProcAttr { return nil }

func platformName() string { return "linux" }

// captureToolName names the capture path actually used on this platform.
func captureToolName() string { return "ffmpeg/x11grab" }
