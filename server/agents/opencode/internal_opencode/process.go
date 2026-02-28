package internal_opencode

import (
	"bufio"
	"fmt"
	"net"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"
)

// PortProcessInfo holds information about a process using a specific port
type PortProcessInfo struct {
	PID      int
	Port     int
	CmdLine  string
	ExecPath string
}

// KillProcessByPort finds and kills the process listening on the given port
// This is a pure Go implementation that doesn't rely on external commands
func KillProcessByPort(port int) error {
	processes, err := FindProcessesByPort(port)
	if err != nil {
		return fmt.Errorf("failed to find processes: %w", err)
	}

	if len(processes) == 0 {
		return fmt.Errorf("no process found listening on port %d", port)
	}

	var lastErr error
	killed := false
	for _, proc := range processes {
		if proc.PID == 0 {
			continue
		}

		// Try graceful termination first (SIGTERM)
		err := syscall.Kill(proc.PID, syscall.SIGTERM)
		if err != nil {
			lastErr = fmt.Errorf("failed to send SIGTERM to PID %d: %w", proc.PID, err)
			// Try SIGKILL as fallback
			err = syscall.Kill(proc.PID, syscall.SIGKILL)
			if err != nil {
				lastErr = fmt.Errorf("failed to send SIGKILL to PID %d: %w", proc.PID, err)
				continue
			}
		}
		killed = true
	}

	if !killed && lastErr != nil {
		return lastErr
	}

	return nil
}

// FindProcessesByPort finds all processes listening on the given port
func FindProcessesByPort(port int) ([]PortProcessInfo, error) {
	// Convert port to hex format used in /proc/net/tcp
	portHex := fmt.Sprintf("%04X", port)

	// Read /proc/net/tcp to find socket inodes for the given port
	inodes, err := findSocketInodes(portHex)
	if err != nil {
		return nil, err
	}

	if len(inodes) == 0 {
		return nil, nil
	}

	// Search through /proc for processes using these inodes
	return findProcessesByInodes(inodes, port)
}

// findSocketInodes reads /proc/net/tcp and /proc/net/tcp6 to find socket inodes
// for connections listening on the given port
func findSocketInodes(portHex string) ([]string, error) {
	var inodes []string

	// Read IPv4 connections
	if data, err := os.ReadFile("/proc/net/tcp"); err == nil {
		inodes = append(inodes, parseNetTCP(string(data), portHex)...)
	}

	// Read IPv6 connections
	if data, err := os.ReadFile("/proc/net/tcp6"); err == nil {
		inodes = append(inodes, parseNetTCP(string(data), portHex)...)
	}

	return inodes, nil
}

// parseNetTCP parses /proc/net/tcp format to find socket inodes for a port
func parseNetTCP(data string, portHex string) []string {
	var inodes []string
	scanner := bufio.NewScanner(strings.NewReader(data))

	// Skip header line
	if scanner.Scan() {
		// header line
	}

	for scanner.Scan() {
		line := scanner.Text()
		fields := strings.Fields(line)
		if len(fields) < 10 {
			continue
		}

		// fields[1] is the local address (IP:PORT in hex)
		// fields[3] is the connection state (0A = LISTEN)
		// fields[9] is the inode
		localAddr := fields[1]
		state := fields[3]
		inode := fields[9]

		// Check if it's a LISTEN state (0A in hex)
		if state != "0A" {
			continue
		}

		// Parse local address to extract port
		parts := strings.Split(localAddr, ":")
		if len(parts) != 2 {
			continue
		}

		// Check if the port matches
		if parts[1] == portHex {
			// Skip inode 0 (not a real socket)
			if inode != "0" {
				inodes = append(inodes, inode)
			}
		}
	}

	return inodes
}

// findProcessesByInodes searches through /proc to find processes using the given socket inodes
func findProcessesByInodes(inodes []string, port int) ([]PortProcessInfo, error) {
	var processes []PortProcessInfo

	entries, err := os.ReadDir("/proc")
	if err != nil {
		return nil, fmt.Errorf("failed to read /proc: %w", err)
	}

	inodeSet := make(map[string]bool)
	for _, inode := range inodes {
		inodeSet[inode] = true
	}

	for _, entry := range entries {
		// Only look at PID directories (numeric names)
		if !entry.IsDir() {
			continue
		}

		pid, err := strconv.Atoi(entry.Name())
		if err != nil {
			continue // Not a PID directory
		}

		// Check if this process has any of the inodes
		if matches, execPath := processHasInodes(pid, inodeSet); matches {
			cmdLine := readCmdLine(pid)
			processes = append(processes, PortProcessInfo{
				PID:      pid,
				Port:     port,
				CmdLine:  cmdLine,
				ExecPath: execPath,
			})
		}
	}

	return processes, nil
}

// processHasInodes checks if a process has file descriptors matching the given inodes
func processHasInodes(pid int, inodeSet map[string]bool) (bool, string) {
	fdDir := fmt.Sprintf("/proc/%d/fd", pid)

	entries, err := os.ReadDir(fdDir)
	if err != nil {
		return false, ""
	}

	found := false
	for _, entry := range entries {
		fdPath := filepath.Join(fdDir, entry.Name())
		linkTarget, err := os.Readlink(fdPath)
		if err != nil {
			continue
		}

		// Check if it's a socket link
		if strings.HasPrefix(linkTarget, "socket:[") {
			inode := strings.TrimPrefix(linkTarget, "socket:[")
			inode = strings.TrimSuffix(inode, "]")

			if inodeSet[inode] {
				found = true
				break
			}
		}
	}

	if !found {
		return false, ""
	}

	// Get the executable path
	execPath, _ := os.Readlink(fmt.Sprintf("/proc/%d/exe", pid))
	return true, execPath
}

// readCmdLine reads the command line for a process
func readCmdLine(pid int) string {
	data, err := os.ReadFile(fmt.Sprintf("/proc/%d/cmdline", pid))
	if err != nil {
		return ""
	}

	// cmdline uses null bytes as separators
	cmdLine := strings.ReplaceAll(string(data), "\x00", " ")
	return strings.TrimSpace(cmdLine)
}

// IsPortInUse checks if a port is currently in use by any process
func IsPortInUse(port int) bool {
	processes, err := FindProcessesByPort(port)
	return err == nil && len(processes) > 0
}

// GetProcessUsingPort returns information about the process using the given port
func GetProcessUsingPort(port int) (*PortProcessInfo, error) {
	processes, err := FindProcessesByPort(port)
	if err != nil {
		return nil, err
	}
	if len(processes) == 0 {
		return nil, fmt.Errorf("no process found listening on port %d", port)
	}
	return &processes[0], nil
}

// GetLocalPort returns the port part of an address
func GetLocalPort(addr string) int {
	_, portStr, err := net.SplitHostPort(addr)
	if err != nil {
		return 0
	}
	port, _ := strconv.Atoi(portStr)
	return port
}
