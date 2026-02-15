package daemon

import (
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"syscall"
	"time"
)

// Daemon represents the keep-alive daemon
type Daemon struct {
	state          *State
	processManager *ProcessManager
	healthChecker  *HealthChecker
	httpServer     *HTTPServer
	port           int
	serverArgs     []string
}

// DualLogger writes to both stdout/stderr and a log file
type DualLogger struct {
	logFile *os.File
	stdout  io.Writer
	stderr  io.Writer
}

// NewDualLogger creates a new dual logger
func NewDualLogger(logPath string) (*DualLogger, error) {
	if logPath == "" || logPath == "no" {
		// No log file, just use stdout/stderr
		return &DualLogger{
			stdout: os.Stdout,
			stderr: os.Stderr,
		}, nil
	}

	logFile, err := os.OpenFile(logPath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
	if err != nil {
		return nil, fmt.Errorf("failed to open log file: %w", err)
	}

	return &DualLogger{
		logFile: logFile,
		stdout:  os.Stdout,
		stderr:  os.Stderr,
	}, nil
}

// Close closes the log file
func (dl *DualLogger) Close() {
	if dl.logFile != nil {
		dl.logFile.Close()
	}
}

// Log prints a timestamped message to both stdout and log file
func (dl *DualLogger) Log(format string, args ...interface{}) {
	timestamp := time.Now().Format("2006-01-02T15:04:05")
	message := fmt.Sprintf("[%s] %s\n", timestamp, fmt.Sprintf(format, args...))

	// Always write to stdout
	fmt.Fprint(dl.stdout, message)

	// Also write to log file if available
	if dl.logFile != nil {
		fmt.Fprint(dl.logFile, message)
	}
}

// GetStdout returns the stdout writer (multiwriter if log file is enabled)
func (dl *DualLogger) GetStdout() io.Writer {
	if dl.logFile != nil {
		return io.MultiWriter(dl.stdout, dl.logFile)
	}
	return dl.stdout
}

// GetStderr returns the stderr writer (multiwriter if log file is enabled)
func (dl *DualLogger) GetStderr() io.Writer {
	if dl.logFile != nil {
		return io.MultiWriter(dl.stderr, dl.logFile)
	}
	return dl.stderr
}

// NewDaemon creates a new daemon instance
func NewDaemon(port int, serverArgs []string) *Daemon {
	state := GlobalState
	return &Daemon{
		state:          state,
		processManager: NewProcessManager(state),
		healthChecker:  NewHealthChecker(state),
		httpServer:     NewHTTPServer(state),
		port:           port,
		serverArgs:     serverArgs,
	}
}

// Run starts the daemon and runs indefinitely
func (d *Daemon) Run(forever bool, logPath string) error {
	// Initialize unified logger
	if err := InitLogger(logPath); err != nil {
		fmt.Fprintf(os.Stderr, "Failed to setup logger: %v\n", err)
	}
	defer CloseLogger()

	// Check if port is already in use - another keep-alive is likely running
	// Skip this check if --forever flag is set
	if !forever && IsPortInUse(d.port) {
		pid := FindPortPID(d.port)
		if pid != "" {
			return fmt.Errorf("port %d is already in use by PID %s - another keep-alive instance may be running", d.port, pid)
		}
		return fmt.Errorf("port %d is already in use - another keep-alive instance may be running", d.port)
	}

	binPath, err := getCurrentExecutablePath()
	if err != nil {
		return fmt.Errorf("resolve executable path: %w", err)
	}

	d.state.SetBinPath(binPath)
	d.state.SetDaemonBinPath(binPath)
	d.state.SetServerPort(d.port)

	// Start HTTP management server
	d.httpServer.Start()

	// Run the main loop
	return d.runLoop()
}

// runLoop is the main daemon loop that manages the server process
func (d *Daemon) runLoop() error {
	for {
		// Before starting, check if there's a newer versioned binary
		currentBin := d.state.GetBinPath()

		if newerBin := FindNewerBinary(currentBin); newerBin != "" {
			Logger("Found newer binary: %s (upgrading from %s)",
				filepath.Base(newerBin), filepath.Base(currentBin))
			d.state.SetBinPath(newerBin)
			currentBin = newerBin
		}

		Logger("Starting ai-critic server on port %d (binary: %s)...",
			d.port, filepath.Base(currentBin))

		// Check if port is already in use - server may already be running (e.g., after exec-restart)
		if IsPortReachable(d.port) {
			Logger("Port %d is already in use, server may already be running (e.g., after exec-restart)", d.port)
			// Server appears to be running, try to connect to it
			// We'll create a dummy cmd to pass to health check - it will detect the running server
			// Find the PID of the process using the port
			pid := FindPortPID(d.port)
			if pid != "" {
				Logger("Found existing server process (PID=%s) on port %d", pid, d.port)
				// Create a command reference to the existing process
				cmd, err := d.reconnectToServer(pid, currentBin)
				if err != nil {
					Logger("Failed to reconnect to server: %v", err)
					Logger("Restarting in %v...", RestartDelay)
					time.Sleep(RestartDelay)
					continue
				}
				setCurrentCommand(cmd)
				d.state.SetServerPID(cmd.Process.Pid)
				d.state.SetStartedAt(time.Now())
				Logger("Reconnected to server (PID=%d)", cmd.Process.Pid)

				// Pause health checks temporarily to give the server time to stabilize after exec-restart
				d.state.PauseHealthChecks(HealthCheckPauseDelay)
				Logger("Health checks paused for %v after exec-restart", HealthCheckPauseDelay)

				// Run health check to monitor the existing server
				exitReason := d.healthChecker.Run(d.port, cmd, currentBin, FindNewerBinary)

				d.state.SetServerPID(0)
				setCurrentCommand(nil)
				d.state.IncrementRestartCount()

				// After health check, loop will restart or continue
				switch exitReason {
				case ExitReasonUpgrade, ExitReasonRestart:
					Logger("%s, restarting immediately...", exitReason)
				case ExitReasonDaemonRestart:
					Logger("Daemon restart requested, stopping and waiting for exec...")
					return nil
				default:
					Logger("Server exited (%s), restarting in %v...", exitReason, RestartDelay)
					time.Sleep(RestartDelay)
				}
				continue
			}
			Logger("Port in use but could not find PID, will attempt to start new server")
		}

		// Start the server process with dual logging
		cmd, err := d.startServerWithLogging(currentBin, d.serverArgs)
		if err != nil {
			Logger("Failed to start server: %v", err)
			d.state.SetServerPID(0)
			Logger("Restarting in %v...", RestartDelay)
			time.Sleep(RestartDelay)
			continue
		}

		pid := cmd.Process.Pid

		setCurrentCommand(cmd)

		// Wait for port to become ready
		ready := d.processManager.WaitForPort(d.port, StartupTimeout, cmd)
		if !ready {
			Logger("ERROR: Server failed to become ready within %v", StartupTimeout)
			d.processManager.KillProcessGroup(cmd)
			d.state.SetServerPID(0)
			setCurrentCommand(nil)
			Logger("Restarting in %v...", RestartDelay)
			time.Sleep(RestartDelay)
			continue
		}

		Logger("Server is ready (PID=%d, port=%d)", pid, d.port)

		// Health check loop (also checks for binary upgrades and restart signals)
		exitReason := d.healthChecker.Run(d.port, cmd, currentBin, FindNewerBinary)

		d.state.SetServerPID(0)
		setCurrentCommand(nil)
		d.state.IncrementRestartCount()

		// After health check exits, check if server is still running (e.g., after exec-restart)
		// If it's still running, don't restart - the exec replaced the process
		if exitReason == ExitReasonUpgrade {
			Logger("Checking if server is still running after upgrade...")
			// Wait a bit for the new server to bind to the port (after exec-restart)
			time.Sleep(2 * time.Second)
			if IsPortReachable(d.port) {
				Logger("Server still running after upgrade (exec-restart succeeded), reconnecting...")
				// Re-run the health check - it will find the running server and monitor it
				// Need to re-check for newer binary and run health check again
				currentBin = d.state.GetBinPath()
				if newerBin := FindNewerBinary(currentBin); newerBin != "" {
					d.state.SetBinPath(newerBin)
					currentBin = newerBin
				}
				// Create a dummy cmd to pass to health check - it will use the existing process
				// Actually, we need to re-detect the process. Let's just continue the loop
				// and let it start fresh but with port check
				Logger("Reconnecting to running server on port %d...", d.port)
				continue
			}
			Logger("Server not reachable after upgrade, starting new process...")
		}

		switch exitReason {
		case ExitReasonUpgrade, ExitReasonRestart:
			Logger("%s, restarting immediately...", exitReason)
		case ExitReasonDaemonRestart:
			Logger("Daemon restart requested, stopping and waiting for exec...")
			return nil
		default:
			Logger("Server exited (%s), restarting in %v...", exitReason, RestartDelay)
			time.Sleep(RestartDelay)
		}
	}
}

// startServerWithLogging starts the server process with dual logging
func (d *Daemon) startServerWithLogging(binPath string, serverArgs []string) (*exec.Cmd, error) {
	// Ensure the binary is executable
	os.Chmod(binPath, 0755)

	cmd := exec.Command(binPath, serverArgs...)
	cmd.Dir, _ = os.Getwd()

	// Create a new process group so we can kill all child processes
	cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}

	// Tee stdout/stderr to both console and log file
	cmd.Stdout = GetLogWriter()
	cmd.Stderr = GetStderrWriter()
	// Close stdin to prevent interactive prompts from hanging the server
	cmd.Stdin = nil

	if err := cmd.Start(); err != nil {
		return nil, fmt.Errorf("failed to start server: %w", err)
	}

	pid := cmd.Process.Pid
	Logger("Server started (PID=%d)", pid)

	d.state.SetServerPID(pid)
	d.state.SetStartedAt(time.Now())

	return cmd, nil
}

// reconnectToServer creates an exec.Cmd reference to an already running server process.
// This is used when the server is already running (e.g., after exec-restart) and we
// want to monitor it without starting a new process.
func (d *Daemon) reconnectToServer(pid string, binPath string) (*exec.Cmd, error) {
	pidNum := 0
	_, err := fmt.Sscanf(pid, "%d", &pidNum)
	if err != nil {
		return nil, fmt.Errorf("invalid PID: %w", err)
	}

	// Create a command that references the existing process
	// We use the findNewerBinary to get the correct binary path
	actualBinPath := binPath
	if newerBin := FindNewerBinary(binPath); newerBin != "" {
		actualBinPath = newerBin
		d.state.SetBinPath(newerBin)
	}

	// Create a command structure - we won't start it, just use it for state tracking
	cmd := exec.Command(actualBinPath, d.serverArgs...)
	cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}

	// Attach to existing process
	cmd.Process = &os.Process{Pid: pidNum}

	Logger("Reconnected to server process (PID=%d, binary=%s)", pidNum, filepath.Base(actualBinPath))
	return cmd, nil
}

// RunKeepAlive is the main entry point for the keep-alive daemon
func RunKeepAlive(port int, forever bool, logPath string, serverArgs []string) error {
	daemon := NewDaemon(port, serverArgs)
	return daemon.Run(forever, logPath)
}
