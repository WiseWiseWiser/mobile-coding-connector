//go:build linux

package daemon

import (
	"bytes"
	"fmt"
	"os"
)

// IsProcessStopped reports whether pid is in job-control stopped state (State: T).
func IsProcessStopped(pid int) bool {
	if pid <= 0 {
		return false
	}
	data, err := os.ReadFile(fmt.Sprintf("/proc/%d/status", pid))
	if err != nil {
		return false
	}
	for _, line := range bytes.Split(data, []byte("\n")) {
		if !bytes.HasPrefix(line, []byte("State:")) {
			continue
		}
		fields := bytes.Fields(line)
		if len(fields) < 2 {
			return false
		}
		state := string(fields[1])
		return len(state) > 0 && state[0] == 'T'
	}
	return false
}