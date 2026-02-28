package common_opencode

import (
	"fmt"
	"net/http"
	"os/exec"
	"syscall"
	"time"
)

// IsCmdAlive checks whether the provided process is still alive.
func IsCmdAlive(cmd *exec.Cmd) bool {
	if cmd == nil || cmd.Process == nil {
		return false
	}
	return syscall.Kill(cmd.Process.Pid, 0) == nil
}

// WaitDone returns a channel closed when cmd exits.
func WaitDone(cmd *exec.Cmd) chan struct{} {
	done := make(chan struct{})
	go func() {
		cmd.Wait()
		close(done)
	}()
	return done
}

// WaitForSessionReady waits for opencode /session endpoint to respond.
func WaitForSessionReady(port int, timeout time.Duration) error {
	url := fmt.Sprintf("http://127.0.0.1:%d/session", port)
	deadline := time.Now().Add(timeout)

	for time.Now().Before(deadline) {
		resp, err := http.Get(url)
		if err == nil {
			resp.Body.Close()
			// Any HTTP response means the process accepted connections.
			return nil
		}
		time.Sleep(500 * time.Millisecond)
	}

	return fmt.Errorf("timeout waiting for server on port %d", port)
}
