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

	"github.com/xhd2015/less-gen/flags"
)

type PortProtectionConfig struct {
	ProtectedPorts map[int]bool `json:"protected_ports"`
}

var help = `Usage: go run ./cmd/safekill <pid> [options]

Kills the process with the given PID.
Protects certain ports from being killed.

Arguments:
  pid   Process ID to kill

Options:
  -9    Send SIGKILL (force kill)
  -15   Send SIGTERM (graceful, default)
  -h, --help   Show this help message

Example:
  go run ./cmd/safekill 12345
  go run ./cmd/safekill 12345 -9
`

func main() {
	if err := run(os.Args[1:]); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}

func run(args []string) error {
	var sig9 bool
	var sig15 bool

	args, err := flags.
		Bool("-9", &sig9).
		Bool("-15", &sig15).
		Help("-h,--help", help).
		Parse(args)
	if err != nil {
		return err
	}

	if len(args) == 0 {
		fmt.Print(help)
		return nil
	}

	signal := syscall.SIGTERM // default
	if sig9 {
		signal = syscall.SIGKILL
	} else if sig15 {
		signal = syscall.SIGTERM
	}

	pidStr := args[0]
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
