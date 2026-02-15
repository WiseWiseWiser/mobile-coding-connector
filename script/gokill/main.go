package main

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"
)

type PortProtectionConfig struct {
	ProtectedPorts map[int]bool `json:"protected_ports"`
}

const help = `Usage: go run ./script/gokill <pid> [-9|-15]

Kills the process with the given PID.
Protects certain ports from being killed.

Arguments:
  pid   Process ID to kill
  -9    Send SIGKILL (force kill)
  -15   Send SIGTERM (graceful, default)

Example:
  go run ./script/gokill 12345
  go run ./script/gokill 12345 -9
`

func main() {
	if err := run(os.Args[1:]); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}

func run(args []string) error {
	if len(args) == 0 || args[0] == "-h" || args[0] == "--help" {
		fmt.Print(help)
		return nil
	}

	signal := syscall.SIGTERM // default

	var pidStr string
	for _, arg := range args {
		if strings.HasPrefix(arg, "-") {
			sig, err := strconv.Atoi(arg[1:])
			if err != nil {
				return fmt.Errorf("invalid signal: %s", arg)
			}
			signal = syscall.Signal(sig)
		} else {
			pidStr = arg
		}
	}

	if pidStr == "" {
		return fmt.Errorf("missing pid argument")
	}

	pid, err := strconv.Atoi(pidStr)
	if err != nil {
		return fmt.Errorf("invalid pid: %s", pidStr)
	}

	return killProcess(pid, signal)
}

func killProcess(pid int, signal syscall.Signal) error {
	if pid == 1 {
		return fmt.Errorf("cannot kill init process (PID 1)")
	}

	err := syscall.Kill(pid, syscall.Signal(0))
	if err != nil {
		if err == syscall.ESRCH {
			return fmt.Errorf("process not found")
		}
		return fmt.Errorf("cannot access process: %v", err)
	}

	ports, err := getPortsForPID(pid)
	if err != nil {
		return fmt.Errorf("failed to get ports for pid: %w", err)
	}

	protected, err := loadProtectedPorts()
	if err != nil {
		return fmt.Errorf("failed to load protected ports: %w", err)
	}

	for _, p := range ports {
		if protected[p] {
			return fmt.Errorf("ask user to restart for you for port %d", p)
		}
	}

	cmd := exec.Command("kill", "-s", strconv.Itoa(int(signal)), strconv.Itoa(pid))
	err = cmd.Run()
	if err != nil {
		return fmt.Errorf("failed to kill process: %v", err)
	}

	fmt.Printf("Killed process %d\n", pid)
	return nil
}

func getPortsForPID(pid int) ([]int, error) {
	cmd := exec.Command("lsof", "-iTCP", "-sTCP:LISTEN", "-n", "-P", "-a", "-p", strconv.Itoa(pid))
	output, err := cmd.Output()
	if err != nil {
		return nil, err
	}

	var ports []int
	lines := strings.Split(string(output), "\n")
	for i, line := range lines {
		if i == 0 || strings.TrimSpace(line) == "" {
			continue
		}
		fields := strings.Fields(line)
		if len(fields) < 9 {
			continue
		}
		nameField := fields[8]
		if idx := strings.LastIndex(nameField, ":"); idx != -1 {
			portStr := nameField[idx+1:]
			port, err := strconv.Atoi(portStr)
			if err == nil && port > 0 {
				ports = append(ports, port)
			}
		}
	}
	return ports, nil
}

func loadProtectedPorts() (map[int]bool, error) {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return nil, err
	}
	credFile := filepath.Join(homeDir, ".ai-critic", "port-protection.json")

	data, err := os.ReadFile(credFile)
	if err != nil {
		if os.IsNotExist(err) {
			return make(map[int]bool), nil
		}
		return nil, err
	}

	var config PortProtectionConfig
	if err := json.Unmarshal(data, &config); err != nil {
		return nil, err
	}

	if config.ProtectedPorts == nil {
		return make(map[int]bool), nil
	}
	return config.ProtectedPorts, nil
}
