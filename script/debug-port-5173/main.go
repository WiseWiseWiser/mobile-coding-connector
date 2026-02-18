package main

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

func main() {
	// Get the project root using git
	gitCmd := exec.Command("git", "rev-parse", "--show-toplevel")
	output, err := gitCmd.Output()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error getting git root: %v\n", err)
		os.Exit(1)
	}

	projectRoot := strings.TrimSpace(string(output))

	// Path to the debug.js script
	debugScript := filepath.Join(projectRoot, "skills", "debug-port-5173", "debug.js")

	// Check if debug.js exists
	if _, err := os.Stat(debugScript); os.IsNotExist(err) {
		fmt.Fprintf(os.Stderr, "Error: debug.js not found at %s\n", debugScript)
		os.Exit(1)
	}

	// Prepare node command
	cmd := exec.Command("node", debugScript)
	cmd.Dir = filepath.Join(projectRoot, "skills", "debug-port-5173")

	// Pass stdin through
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	// Run the command
	if err := cmd.Run(); err != nil {
		if exitError, ok := err.(*exec.ExitError); ok {
			os.Exit(exitError.ExitCode())
		}
		fmt.Fprintf(os.Stderr, "Error running debug script: %v\n", err)
		os.Exit(1)
	}
}
