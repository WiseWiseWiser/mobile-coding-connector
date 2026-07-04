package daemon

import (
	"os/exec"
	"strconv"
	"strings"
	"syscall"
)

// TestExported_OccupierDead reports whether a port-occupier PID is gone or defunct (zombie).
// External SIGKILL/SIGTERM from keep-alive leaves zombie children until the parent reaps;
// zombies still answer kill(pid,0) but show state Z in ps.
func TestExported_OccupierDead(pid int) bool {
	if pid <= 0 {
		return true
	}
	out, err := exec.Command("ps", "-p", strconv.Itoa(pid), "-o", "state=").Output()
	if err != nil {
		return true
	}
	state := strings.TrimSpace(string(out))
	if state == "" || state == "Z" {
		return true
	}
	return syscall.Kill(pid, 0) != nil
}