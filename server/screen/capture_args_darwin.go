//go:build darwin

package screen

import (
	"fmt"
	"os"
	"os/exec"
	"regexp"
	"strconv"
	"sync"
	"syscall"
)

var (
	deviceOnce sync.Once
	deviceIdx  string
	deviceErr  error
)

var captureScreenRe = regexp.MustCompile(`\[(\d+)\]\s+Capture screen`)

// captureArgs builds the ffmpeg invocation for the macOS screen device.
// -capture_cursor 0 keeps the pointer out of the video: the browser draws the
// real pointer itself at 30 Hz, which looks far smoother than the video rate.
func captureArgs(maxWidth, fps int) ([]string, error) {
	dev, err := screenDevice()
	if err != nil {
		return nil, err
	}
	return []string{
		"-hide_banner", "-loglevel", "error",
		"-fflags", "nobuffer",
		"-f", "avfoundation",
		"-capture_cursor", "0",
		"-framerate", strconv.Itoa(fps),
		"-i", dev,
		"-an",
		"-vf", fmt.Sprintf("scale=%d:-2:flags=bilinear", maxWidth),
		"-q:v", "3",
		"-f", "mjpeg",
		"-flush_packets", "1",
		"pipe:1",
	}, nil
}

// screenDevice picks the AVFoundation screen input. Device 0 is usually the
// FaceTime camera, so the screen must be discovered rather than assumed.
func screenDevice() (string, error) {
	deviceOnce.Do(func() {
		if override := os.Getenv("AI_CRITIC_SCREEN_DEVICE"); override != "" {
			deviceIdx = override
			return
		}
		out, err := exec.Command("ffmpeg", "-hide_banner",
			"-f", "avfoundation", "-list_devices", "true", "-i", "").CombinedOutput()
		if err != nil && len(out) == 0 {
			deviceErr = fmt.Errorf("enumerate capture devices: %w", err)
			return
		}
		m := captureScreenRe.FindSubmatch(out)
		if m == nil {
			deviceErr = fmt.Errorf("no screen capture device reported by ffmpeg (is Screen Recording permission granted?)")
			return
		}
		deviceIdx = string(m[1]) + ":none"
	})
	if deviceErr != nil {
		return "", deviceErr
	}
	return deviceIdx, nil
}

func sysProcAttr() *syscall.SysProcAttr { return nil }

func platformName() string { return "darwin" }

// captureToolName names the capture path actually used on this platform.
func captureToolName() string { return "screencap/sck" }
