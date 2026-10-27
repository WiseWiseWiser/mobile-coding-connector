//go:build darwin

package screen

import (
	"log"
	"os"
	"os/exec"
	"strconv"
	"strings"
)

// reapStaleCapture kills orphaned screen captures left behind by an earlier
// ai-critic process (crash, hard kill, or a failed upgrade). A capture holds
// the screen device and keeps capturing the user's desktop, so such an orphan
// both leaks a capture and can starve the next one.
//
// Only reparented processes (ppid 1) that carry our exact capture flags are
// touched; a capture owned by a live process is left alone.
func reapStaleCapture() {
	out, err := exec.Command("ps", "-axo", "pid=,ppid=,command=").Output()
	if err != nil {
		return
	}
	self := os.Getpid()
	for _, line := range strings.Split(string(out), "\n") {
		fields := strings.Fields(line)
		if len(fields) < 4 {
			continue
		}
		pid, err := strconv.Atoi(fields[0])
		if err != nil || pid <= 1 || pid == self {
			continue
		}
		if fields[1] != "1" {
			continue
		}
		cmd := strings.Join(fields[2:], " ")
		ffmpegCapture := strings.Contains(cmd, "-f avfoundation") && strings.Contains(cmd, "-capture_cursor 0")
		screencap := strings.Contains(cmd, screencapName) && strings.Contains(cmd, "--width")
		if !ffmpegCapture && !screencap {
			continue
		}
		proc, err := os.FindProcess(pid)
		if err != nil {
			continue
		}
		// A capture blocked inside AVFoundation ignores SIGTERM, so SIGKILL it.
		if err := proc.Kill(); err != nil {
			continue
		}
		log.Printf("[screen] reaped stale capture pid=%d (orphaned and holding the screen device)", pid)
	}
}
