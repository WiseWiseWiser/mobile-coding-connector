//go:build darwin

package screen

import (
	"context"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"

	"github.com/xhd2015/ai-critic/server/config"
)

// screencapName is the ScreenCaptureKit helper shipped alongside the server.
const screencapName = "ai-critic-screencap"

// startPlatformSource captures with the ScreenCaptureKit helper. macOS 26 no
// longer feeds the legacy AVFoundation screen input: ffmpeg still starts and
// still reports the real frame size, but every frame after the first is a copy
// of the first. The helper exists because that failure is invisible from the
// outside — the capture looks healthy while the mirror stays frozen.
//
// ffmpeg stays as a fallback so an older macOS (or a source build without the
// helper) still mirrors, only without the fix.
func startPlatformSource(ctx context.Context, maxWidth, fps, quality int) (source, error) {
	if bin := screencapHelper(); bin != "" {
		log.Printf("[screen] capturing with %s (ScreenCaptureKit, %dx%s at %d fps)", bin, maxWidth, "auto", fps)
		return startProcess(ctx, "screencap", bin, screencapArgs(maxWidth, fps, quality))
	}
	log.Printf("[screen] %s not found: falling back to ffmpeg avfoundation, which freezes after the first frame on macOS 26+", screencapName)
	args, err := captureArgs(maxWidth, fps)
	if err != nil {
		return nil, err
	}
	return startFFmpeg(ctx, args)
}

// screencapHelper locates the helper: an explicit override, the app bundle or
// binary directory, the data directory used by dev installs, then PATH.
func screencapHelper() string {
	if override := os.Getenv("AI_CRITIC_SCREENCAP"); override != "" {
		if isExecutable(override) {
			return override
		}
		log.Printf("[screen] AI_CRITIC_SCREENCAP=%s is not executable, ignoring it", override)
	}
	var candidates []string
	if exe, err := os.Executable(); err == nil {
		if resolved, err := filepath.EvalSymlinks(exe); err == nil {
			exe = resolved
		}
		candidates = append(candidates, filepath.Join(filepath.Dir(exe), screencapName))
	}
	candidates = append(candidates, filepath.Join(config.DataDir, "bin", screencapName))
	for _, candidate := range candidates {
		if isExecutable(candidate) {
			return candidate
		}
	}
	if path, err := exec.LookPath(screencapName); err == nil {
		return path
	}
	return ""
}

func isExecutable(path string) bool {
	info, err := os.Stat(path)
	return err == nil && !info.IsDir() && info.Mode()&0o111 != 0
}

func screencapArgs(maxWidth, fps, quality int) []string {
	return []string{
		"--width", strconv.Itoa(maxWidth),
		"--fps", strconv.Itoa(fps),
		"--quality", strconv.FormatFloat(float64(quality)/100, 'f', 2, 64),
	}
}
