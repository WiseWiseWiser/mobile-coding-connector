package daemon

import (
	"bufio"
	"fmt"
	"net/http"
	"os"
	"os/exec"
	"strings"
	"syscall"
	"time"

	"github.com/xhd2015/lifelog-private/ai-critic/server/config"
	"github.com/xhd2015/lifelog-private/ai-critic/server/sse"
)

const daemonShutdownDrainDelay = 200 * time.Millisecond

// CallShutdownEndpoint calls the server's shutdown endpoint with auth.
// Returns true if the request was successful.
func CallShutdownEndpoint() bool {
	token, err := loadFirstToken()
	if err != nil {
		Logger("Failed to load auth token: %v", err)
		return false
	}

	port := config.DefaultServerPort
	url := fmt.Sprintf("http://localhost:%d/api/shutdown", port)

	req, err := http.NewRequest(http.MethodPost, url, nil)
	if err != nil {
		Logger("Failed to create shutdown request: %v", err)
		return false
	}

	if token != "" {
		req.AddCookie(&http.Cookie{
			Name:  "ai-critic-token",
			Value: token,
		})
	}

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		Logger("Failed to call shutdown endpoint: %v", err)
		return false
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusOK {
		Logger("Shutdown endpoint returned success")
		return true
	}

	Logger("Shutdown endpoint returned status: %d", resp.StatusCode)
	return false
}

func (s *HTTPServer) shutdownDaemonForExec(stopManagedServer bool, sseWriter *sse.Writer) error {
	if s.state.RequestDaemonShutdown() {
		logShutdownMessage(sseWriter, "Daemon shutdown requested; draining background loops...")
	} else {
		logShutdownMessage(sseWriter, "Daemon shutdown was already requested; continuing...")
	}

	time.Sleep(daemonShutdownDrainDelay)

	if !stopManagedServer {
		logShutdownMessage(sseWriter, "Managed server will remain running during daemon exec-replace")
		return nil
	}

	return s.stopManagedServerForDaemonExec(sseWriter)
}

func (s *HTTPServer) stopManagedServerForDaemonExec(sseWriter *sse.Writer) error {
	cmd := getCurrentCommand()
	if cmd == nil || cmd.Process == nil {
		logShutdownMessage(sseWriter, "No managed server process is currently attached")
		return nil
	}

	pid := cmd.Process.Pid
	logShutdownMessage(sseWriter, fmt.Sprintf("Stopping server PID %d before daemon exec...", pid))

	done := make(chan struct{}, 1)
	go func() {
		cmd.Process.Wait()
		close(done)
	}()

	if CallShutdownEndpoint() {
		logShutdownMessage(sseWriter, "Graceful shutdown request sent")
		select {
		case <-done:
			logShutdownMessage(sseWriter, "Server stopped gracefully")
		case <-time.After(30 * time.Second):
			logShutdownMessage(sseWriter, "Graceful shutdown timeout, force killing...")
			killProcess(cmd)
			select {
			case <-done:
				logShutdownMessage(sseWriter, "Server force stopped")
			case <-time.After(5 * time.Second):
				logShutdownMessage(sseWriter, "Warning: server may still be running")
			}
		}
	} else {
		logShutdownMessage(sseWriter, "Shutdown endpoint unavailable, using direct kill")
		killProcess(cmd)
		select {
		case <-done:
			logShutdownMessage(sseWriter, "Server stopped")
		case <-time.After(5 * time.Second):
			logShutdownMessage(sseWriter, "Warning: server may still be running")
		}
	}

	setCurrentCommand(nil)
	s.state.SetServerPID(0)
	return nil
}

// killProcess kills a process by PID.
func killProcess(cmd *exec.Cmd) {
	if cmd.Process == nil {
		return
	}

	pgid, err := syscall.Getpgid(cmd.Process.Pid)
	if err != nil {
		Logger("Warning: could not get process group, falling back to process kill")
		cmd.Process.Signal(syscall.SIGTERM)
		time.Sleep(3 * time.Second)
		cmd.Process.Signal(syscall.SIGKILL)
		return
	}

	Logger("Killing process group %d", pgid)
	syscall.Kill(-pgid, syscall.SIGKILL)
}

// loadFirstToken reads the first non-empty line from the credentials file.
func loadFirstToken() (string, error) {
	f, err := os.Open(config.CredentialsFile)
	if err != nil {
		return "", err
	}
	defer f.Close()

	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line != "" {
			return line, nil
		}
	}
	return "", scanner.Err()
}

func logShutdownMessage(sseWriter *sse.Writer, message string) {
	Logger("[shutdown] %s", message)
	if sseWriter != nil {
		sseWriter.SendLog(message)
	}
}
