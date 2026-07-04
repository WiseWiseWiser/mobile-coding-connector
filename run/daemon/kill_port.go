package daemon

import (
	"strconv"
	"strings"
	"syscall"
	"time"

	"os/exec"
)

// KillListenersOnPort sends SIGTERM to processes listening on port, then SIGKILL if needed.
func KillListenersOnPort(port int) {
	pids := listenerPIDs(port)
	if len(pids) == 0 {
		return
	}

	selfPID := syscall.Getpid()
	for _, pid := range pids {
		if pid <= 0 || pid == selfPID {
			continue
		}
		Logger("Kill-existing: sending SIGTERM to PID %d on port %d", pid, port)
		_ = syscall.Kill(pid, syscall.SIGTERM)
	}

	time.Sleep(300 * time.Millisecond)

	for _, pid := range pids {
		if pid <= 0 || pid == selfPID {
			continue
		}
		if syscall.Kill(pid, 0) == nil {
			Logger("Kill-existing: sending SIGKILL to PID %d on port %d", pid, port)
			_ = syscall.Kill(pid, syscall.SIGKILL)
		}
	}
}

// listenerPIDs returns PIDs with a TCP listener on the given port.
func listenerPIDs(port int) []int {
	cmd := exec.Command("lsof", "-nP", "-iTCP:"+strconv.Itoa(port), "-sTCP:LISTEN", "-t")
	output, err := cmd.Output()
	if err != nil || len(output) == 0 {
		return nil
	}
	var pids []int
	for _, pidStr := range strings.Fields(strings.TrimSpace(string(output))) {
		pid, err := strconv.Atoi(pidStr)
		if err != nil || pid <= 0 {
			continue
		}
		pids = append(pids, pid)
	}
	return pids
}