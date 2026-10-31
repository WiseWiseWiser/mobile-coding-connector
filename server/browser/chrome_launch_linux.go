//go:build linux

package browser

import (
	"context"
	"fmt"
	"os/exec"
	"strings"
)

// Preserve the existing Linux launch behavior; macOS has stricter profile routing.
func openChromeProfile(ctx context.Context, url string, newWindow bool) error {
	if newWindow {
		for _, bin := range []string{"google-chrome", "google-chrome-stable", "chromium"} {
			if _, err := exec.LookPath(bin); err != nil {
				continue
			}
			cmd := exec.Command(bin, "--new-window", url)
			if err := cmd.Start(); err != nil {
				return fmt.Errorf("open a new Chrome window: %w", err)
			}
			go func() { _ = cmd.Wait() }()
			return nil
		}
	}
	cmd := exec.CommandContext(ctx, "open", "-a", "Google Chrome", url)
	out, err := cmd.CombinedOutput()
	if err != nil {
		if msg := strings.TrimSpace(string(out)); msg != "" {
			return fmt.Errorf("%w: %s", err, msg)
		}
		return err
	}
	return nil
}
