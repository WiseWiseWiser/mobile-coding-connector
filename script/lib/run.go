package lib

import (
	"errors"
	"fmt"
	"net"
	"os/exec"
	"runtime"
	"strconv"
	"strings"
	"syscall"
	"time"
)

var ErrNoProcessOnPort = errors.New("no process found on port")

func CheckPort(port int) bool {
	conn, err := net.DialTimeout("tcp", fmt.Sprintf("localhost:%d", port), 1*time.Second)
	if err != nil {
		return false
	}
	conn.Close()
	return true
}

func GetPidOnPort(port int) (int, error) {
	portStr := fmt.Sprintf("%d", port)

	switch runtime.GOOS {
	case "darwin", "linux":
		cmd := exec.Command("lsof", "-ti", fmt.Sprintf("tcp:%s", portStr), "-sTCP:LISTEN")
		out, err := cmd.Output()
		if err == nil {
			pidStr := strings.TrimSpace(string(out))
			if idx := strings.IndexByte(pidStr, '\n'); idx > 0 {
				pidStr = pidStr[:idx]
			}
			if pidStr == "" {
				return 0, fmt.Errorf("%w: %d", ErrNoProcessOnPort, port)
			}
			pid, err := strconv.Atoi(pidStr)
			if err != nil {
				return 0, fmt.Errorf("failed to parse PID '%s': %w", pidStr, err)
			}
			return pid, nil
		}
		if runtime.GOOS == "linux" {
			cmd = exec.Command("ss", "-tlnp", fmt.Sprintf("sport = :%s", portStr))
			out, err = cmd.Output()
			if err != nil {
				return 0, fmt.Errorf("%w: %d", ErrNoProcessOnPort, port)
			}
			for _, line := range strings.Split(string(out), "\n") {
				if idx := strings.Index(line, "pid="); idx >= 0 {
					rest := line[idx+4:]
					if end := strings.IndexAny(rest, ",) \t\n"); end > 0 {
						pid, err := strconv.Atoi(rest[:end])
						if err != nil {
							return 0, fmt.Errorf("failed to parse PID '%s': %w", rest[:end], err)
						}
						return pid, nil
					}
				}
			}
		}
		return 0, fmt.Errorf("%w: %d", ErrNoProcessOnPort, port)
	default:
		return 0, fmt.Errorf("unsupported platform: %s", runtime.GOOS)
	}
}

// KillPortPid kills any process listening on the given port and waits for it to be free.
// Returns the killed PID, or 0 if no process was found.
func KillPortPid(port int) (int, error) {
	pid, err := GetPidOnPort(port)
	if err != nil {
		if errors.Is(err, ErrNoProcessOnPort) {
			return 0, nil
		}
		return 0, fmt.Errorf("failed to find process on port %d: %w", port, err)
	}

	if err := syscall.Kill(pid, syscall.SIGKILL); err != nil {
		return pid, fmt.Errorf("failed to kill process %d on port %d: %w", pid, port, err)
	}

	deadline := time.Now().Add(5 * time.Second)
	for time.Now().Before(deadline) {
		if !CheckPort(port) {
			time.Sleep(200 * time.Millisecond)
			return pid, nil
		}
		time.Sleep(100 * time.Millisecond)
	}

	return pid, fmt.Errorf("port %d is still in use after killing process %d", port, pid)
}
