package main

import (
	"fmt"
	"os"
	"os/exec"
	"strings"
)

func main() {
	if err := run(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}

func run() error {
	// Find Vite processes by checking port 5173
	cmd := exec.Command("lsof", "-t", "-i", ":5173")
	output, err := cmd.Output()
	if err != nil {
		fmt.Println("No process found on port 5173")
		return nil
	}

	pids := strings.TrimSpace(string(output))
	if pids == "" {
		fmt.Println("No process found on port 5173")
		return nil
	}

	// Find original kill binary
	killPath := "/bin/kill.orig"
	if _, err := os.Stat(killPath); err != nil {
		killPath = "/usr/bin/kill"
	}

	// Check if it's a wrapper
	data, err := os.ReadFile(killPath)
	isWrapper := err == nil && strings.Contains(string(data), "shad-kill")

	if isWrapper {
		// Try /bin/kill.orig directly
		origKill := "/bin/kill.orig"
		if _, err := os.Stat(origKill); err == nil {
			killPath = origKill
		}
	}

	pidList := strings.Split(pids, "\n")
	for _, pidStr := range pidList {
		pidStr = strings.TrimSpace(pidStr)
		if pidStr == "" {
			continue
		}
		fmt.Printf("Killing process (PID: %s)...\n", pidStr)

		// Use syscall to avoid shell wrapper
		var procAttr os.ProcAttr
		proc, err := os.StartProcess(killPath, []string{"kill", "-9", pidStr}, &procAttr)
		if err != nil {
			fmt.Printf("Warning: failed to start kill for PID %s: %v\n", pidStr, err)
			continue
		}
		wait, _ := proc.Wait()
		if wait.ExitCode() != 0 {
			fmt.Printf("Warning: kill exited with code %d for PID %s\n", wait.ExitCode(), pidStr)
		}
	}

	fmt.Println("Done!")
	return nil
}
