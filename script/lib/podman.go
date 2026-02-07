package lib

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"time"
)

const (
	ContainerName  = "ai-critic-sandbox"
	ContainerImage = "docker.io/library/debian:bookworm-slim"
)

// EnsurePodman checks that podman is installed and the machine is running.
func EnsurePodman() error {
	fmt.Println("=== Checking podman ===")

	// Check if podman is installed
	if _, err := exec.LookPath("podman"); err != nil {
		return fmt.Errorf("podman is not installed. Please install it first:\n  macOS: brew install podman\n  Linux: https://podman.io/docs/installation")
	}

	// Check if the machine needs to be started
	var infoBuf bytes.Buffer
	infoCmd := exec.Command("podman", "machine", "info")
	infoCmd.Stdout = &infoBuf
	infoCmd.Stderr = &infoBuf
	if err := infoCmd.Run(); err != nil {
		fmt.Println("No podman machine found. Initializing...")
		if err := RunVerbose("podman", "machine", "init"); err != nil {
			return fmt.Errorf("podman machine init failed: %v", err)
		}
		fmt.Println("Starting podman machine...")
		if err := RunVerbose("podman", "machine", "start"); err != nil {
			return fmt.Errorf("podman machine start failed: %v", err)
		}
		fmt.Println("Podman machine started.")
		return nil
	}

	// Machine exists — check if server is reachable
	var versionBuf bytes.Buffer
	versionCmd := exec.Command("podman", "version", "--format", "{{.Server.Version}}")
	versionCmd.Stdout = &versionBuf
	versionCmd.Stderr = &versionBuf
	if err := versionCmd.Run(); err != nil {
		fmt.Println("Podman machine is not running. Starting...")
		if startErr := RunVerbose("podman", "machine", "start"); startErr != nil {
			return fmt.Errorf("podman machine start failed: %v (original error: %v)", startErr, err)
		}
		fmt.Println("Podman machine started.")
		return nil
	}

	fmt.Printf("Podman ready (server version: %s)\n", strings.TrimSpace(versionBuf.String()))

	// Verify network connectivity inside the VM
	if err := checkVMNetwork(); err != nil {
		return err
	}

	return nil
}

const networkCheckTimeout = 15 * time.Second

// checkVMNetwork verifies the podman VM can reach the internet.
// If not, it restarts the machine (a common fix for stale network bridges).
func checkVMNetwork() error {
	fmt.Print("Checking VM network connectivity... ")

	ok, err := probeVMNetwork()
	if err == nil && ok {
		fmt.Println("OK")
		return nil
	}

	// Network unreachable — restart the machine to fix stale bridge
	fmt.Println("FAILED (no connectivity)")
	fmt.Println("Restarting podman machine to fix networking...")
	if err := RunVerbose("podman", "machine", "stop"); err != nil {
		return fmt.Errorf("podman machine stop failed: %v", err)
	}
	if err := RunVerbose("podman", "machine", "start"); err != nil {
		return fmt.Errorf("podman machine start failed: %v", err)
	}

	// Re-verify after restart
	fmt.Print("Re-checking VM network connectivity... ")
	ok2, err2 := probeVMNetwork()
	if err2 != nil || !ok2 {
		return fmt.Errorf("podman VM still has no network connectivity after restart. Please check your host network and try again")
	}

	fmt.Println("OK")
	return nil
}

// probeVMNetwork tests whether the podman VM has network connectivity
// by pinging a well-known DNS server (8.8.8.8) with a timeout.
// Returns (true, nil) if the VM can reach the network.
func probeVMNetwork() (bool, error) {
	ctx, cancel := context.WithTimeout(context.Background(), networkCheckTimeout)
	defer cancel()

	// Use ping to 8.8.8.8 (Google DNS) — always available, no dependency on specific services
	// also curl:
	//  "curl", "-sI", "-o", "/dev/null", "-w", "%{http_code}", "--connect-timeout", "5", "https://registry-1.docker.io/v2/"
	checkCmd := exec.CommandContext(ctx, "podman", "machine", "ssh", "--",
		"ping", "-c", "1", "-W", "5", "8.8.8.8")
	var out bytes.Buffer
	checkCmd.Stdout = &out
	checkCmd.Stderr = &out
	err := checkCmd.Run()

	if ctx.Err() == context.DeadlineExceeded {
		return false, fmt.Errorf("network check timed out after %v", networkCheckTimeout)
	}

	if err != nil {
		return false, err
	}
	return true, nil
}

// PodmanArch returns the architecture of the podman VM (e.g. "amd64", "arm64").
func PodmanArch() (string, error) {
	var buf bytes.Buffer
	c := exec.Command("podman", "info", "--format", "{{.Host.Arch}}")
	c.Stdout = &buf
	c.Stderr = &buf
	if err := c.Run(); err != nil {
		return "", fmt.Errorf("failed to detect podman architecture: %v", err)
	}
	arch := strings.TrimSpace(buf.String())
	if arch == "" {
		return "", fmt.Errorf("podman returned empty architecture")
	}
	return arch, nil
}

// RunVerbose runs a command with stdout/stderr connected to the terminal.
func RunVerbose(name string, args ...string) error {
	c := exec.Command(name, args...)
	c.Stdout = os.Stdout
	c.Stderr = os.Stderr
	return c.Run()
}
