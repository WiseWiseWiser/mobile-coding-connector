package main

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"strings"
	"syscall"

	"github.com/xhd2015/less-gen/flags"
	"github.com/xhd2015/lifelog-private/ai-critic/script/lib"
	"github.com/xhd2015/xgo/support/cmd"
)

const containerPort = 8899

var help = `
Usage: go run ./script/sandbox/fresh-setup [options]

Builds the frontend and Go server as a single Linux binary,
then runs it inside a podman container exposed on port 8899.

Options:
  --arch ARCH   Target architecture: auto, amd64, arm64 (default: auto)
  -h, --help    Show this help message

Steps:
  1. npm install + npm run build (frontend)
  2. GOOS=linux GOARCH=<arch> go build (server with embedded frontend)
  3. podman create + podman cp + podman start, port 8899 exported
`

func main() {
	var archFlag string
	_, err := flags.
		String("--arch", &archFlag).
		Help("-h,--help", help).
		Parse(os.Args[1:])
	if err != nil {
		fmt.Fprintf(os.Stderr, "%v\n", err)
		os.Exit(1)
	}
	if archFlag == "" {
		archFlag = "auto"
	}

	if err := run(archFlag); err != nil {
		fmt.Fprintf(os.Stderr, "%v\n", err)
		os.Exit(1)
	}
}

func run(archFlag string) error {
	// Step 0: Ensure podman is available and the machine is running
	if err := lib.EnsurePodman(); err != nil {
		return err
	}

	// Resolve target architecture
	goarch, err := resolveArch(archFlag)
	if err != nil {
		return err
	}

	// Abort if target arch differs from VM arch — Go binaries crash under
	// Rosetta/QEMU x86_64 emulation (SIGSEGV in epoll netpoll).
	vmArch, vmErr := lib.PodmanArch()
	if vmErr == nil && vmArch != goarch {
		return fmt.Errorf(
			"target arch %q differs from podman VM arch %q.\n"+
				"  Go binaries crash under Rosetta/QEMU emulation (SIGSEGV in netpoll_epoll).\n"+
				"  Use --arch %s or --arch auto for local testing.\n"+
				"  For amd64 builds, use a real amd64 machine or CI/CD.",
			goarch, vmArch, vmArch,
		)
	}

	// Step 1: Build frontend
	fmt.Println("\n=== Step 1: Building frontend ===")

	if _, err := os.Stat("ai-critic-react/node_modules"); err != nil {
		fmt.Println("node_modules not found, running npm install...")
		if err := cmd.Debug().Dir("ai-critic-react").Run("npm", "install"); err != nil {
			return fmt.Errorf("npm install failed: %v", err)
		}
	}

	if err := cmd.Debug().Dir("ai-critic-react").Run("npm", "run", "build"); err != nil {
		return fmt.Errorf("frontend build failed: %v", err)
	}
	fmt.Println("Frontend build complete.")

	// Step 2: Cross-compile Go server
	binaryPath := fmt.Sprintf("/tmp/ai-critic-linux-%s", goarch)
	fmt.Printf("\n=== Step 2: Cross-compiling Go server for linux/%s ===\n", goarch)

	if err := lib.BuildServer(lib.BuildServerOptions{
		Output: binaryPath,
		GOOS:   "linux",
		GOARCH: goarch,
	}); err != nil {
		return err
	}

	// Step 3: Create container and copy binary in
	// (Volume mounts from /tmp don't work on macOS because podman runs in a VM;
	//  use podman cp to transfer the binary into the container instead.)
	fmt.Println("\n=== Step 3: Setting up podman container ===")

	// Create a stopped container with the target platform.
	// Bind-mount local cache dirs so apt packages and downloaded files persist
	// across container recreations and are visible on the host.
	// Uses os.UserCacheDir() so the system can reclaim space when storage is low.
	systemCacheDir, err := os.UserCacheDir()
	if err != nil {
		return fmt.Errorf("failed to get system cache directory: %v", err)
	}
	cacheBase := systemCacheDir + "/ai-critic"
	aptArchivesDir := cacheBase + "/apt-archives"
	aptListsDir := cacheBase + "/apt-lists"
	downloadsDir := cacheBase + "/downloads"
	for _, dir := range []string{aptArchivesDir, aptListsDir, downloadsDir} {
		if err := os.MkdirAll(dir, 0755); err != nil {
			return fmt.Errorf("failed to create cache dir %s: %v", dir, err)
		}
	}
	fmt.Printf("Cache directory: %s\n", cacheBase)

	// Ensure credentials file exists for volume mount (create empty if missing)
	credentialsFile, err := credentialsFilePath()
	if err != nil {
		return fmt.Errorf("failed to resolve credentials file path: %v", err)
	}
	if _, err := os.Stat(credentialsFile); os.IsNotExist(err) {
		if err := os.WriteFile(credentialsFile, nil, 0600); err != nil {
			return fmt.Errorf("failed to create credentials file: %v", err)
		}
	}
	fmt.Printf("Credentials file: %s\n", credentialsFile)
	const containerCredentialsFile = "/root/.ai-critic/server-credentials"

	// Ensure encryption key files exist for volume mount (create empty if missing)
	encKeyFile, err := encKeyFilePath()
	if err != nil {
		return fmt.Errorf("failed to resolve encryption key file path: %v", err)
	}
	encKeyPubFile := encKeyFile + ".pub"
	for _, f := range []string{encKeyFile, encKeyPubFile} {
		if _, err := os.Stat(f); os.IsNotExist(err) {
			if err := os.WriteFile(f, nil, 0600); err != nil {
				return fmt.Errorf("failed to create encryption key file %s: %v", f, err)
			}
		}
	}
	fmt.Printf("Encryption key file: %s\n", encKeyFile)
	const containerEncKeyFile = "/root/.ai-critic/enc-key"
	const containerEncKeyPubFile = "/root/.ai-critic/enc-key.pub"

	// Ensure domains file exists for volume mount (create empty if missing)
	domainsFile, err := domainsFilePath()
	if err != nil {
		return fmt.Errorf("failed to resolve domains file path: %v", err)
	}
	if _, err := os.Stat(domainsFile); os.IsNotExist(err) {
		if err := os.WriteFile(domainsFile, nil, 0644); err != nil {
			return fmt.Errorf("failed to create domains file: %v", err)
		}
	}
	fmt.Printf("Domains file: %s\n", domainsFile)
	const containerDomainsFile = "/root/.ai-critic/server-domains.json"

	// Remove any existing container (ignore errors if it doesn't exist)
	fmt.Println("Removing old container (if any)...")
	_ = lib.RunVerbose("podman", "rm", "-f", lib.ContainerName)

	platform := fmt.Sprintf("linux/%s", goarch)
	fmt.Printf("Creating container (platform: %s)...\n", platform)
	createArgs := []string{
		"create",
		"--name", lib.ContainerName,
		"--platform", platform,
		"-w", "/root",
		"-v", aptArchivesDir + ":/var/cache/apt/archives",
		"-v", aptListsDir + ":/var/lib/apt/lists",
		"-v", downloadsDir + ":/tmp/downloads",
		"-v", credentialsFile + ":" + containerCredentialsFile,
		"-v", encKeyFile + ":" + containerEncKeyFile,
		"-v", encKeyPubFile + ":" + containerEncKeyPubFile,
		"-v", domainsFile + ":" + containerDomainsFile,
		"-p", fmt.Sprintf("%d:%d", containerPort, containerPort),
		lib.ContainerImage,
		"/usr/local/bin/ai-critic", "--port", fmt.Sprintf("%d", containerPort),
		"--credentials-file", containerCredentialsFile,
		"--enc-key-file", containerEncKeyFile,
		"--domains-file", containerDomainsFile,
	}
	if err := lib.RunVerbose("podman", createArgs...); err != nil {
		return fmt.Errorf("failed to create container: %v", err)
	}

	// Copy binary into the container
	fmt.Println("Copying binary into container...")
	if err := lib.RunVerbose("podman", "cp", binaryPath, lib.ContainerName+":/usr/local/bin/ai-critic"); err != nil {
		return fmt.Errorf("failed to copy binary into container: %v", err)
	}

	// Step 4: Start the container and follow logs
	fmt.Printf("\nStarting container (platform: %s)...\nServer will be available at http://localhost:%d\n\n", platform, containerPort)
	if err := lib.RunVerbose("podman", "start", lib.ContainerName); err != nil {
		return fmt.Errorf("failed to start container: %v", err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)
	go func() {
		<-sigChan
		fmt.Println("\nShutting down container...")
		cancel()
	}()

	// Follow container logs until it exits or user interrupts
	logsCmd := exec.CommandContext(ctx, "podman", "logs", "-f", lib.ContainerName)
	logsCmd.Stdout = os.Stdout
	logsCmd.Stderr = os.Stderr
	if err := logsCmd.Start(); err != nil {
		return fmt.Errorf("failed to follow container logs: %v", err)
	}

	done := make(chan error, 1)
	go func() {
		done <- logsCmd.Wait()
	}()

	select {
	case <-ctx.Done():
		_ = exec.Command("podman", "stop", lib.ContainerName).Run()
	case err := <-done:
		if err != nil && ctx.Err() == nil {
			return fmt.Errorf("container exited with error: %v", err)
		}
	}

	return nil
}

// resolveArch resolves the target architecture from the --arch flag.
// "auto" detects the podman VM's architecture, otherwise uses the specified value.
func resolveArch(archFlag string) (string, error) {
	if archFlag == "auto" {
		return lib.PodmanArch()
	}
	switch archFlag {
	case "amd64", "arm64":
		return archFlag, nil
	default:
		return "", fmt.Errorf("unsupported architecture: %s (supported: auto, amd64, arm64)", archFlag)
	}
}

// scriptDir returns the absolute path to the directory containing this script,
// derived from the git repository root.
func scriptDir() (string, error) {
	out, err := exec.Command("git", "rev-parse", "--show-toplevel").Output()
	if err != nil {
		return "", fmt.Errorf("failed to get git repo root: %v", err)
	}
	repoRoot := strings.TrimSpace(string(out))
	return filepath.Join(repoRoot, "script", "sandbox", "fresh-setup"), nil
}

// credentialsFilePath returns the absolute path to the credentials file,
// located alongside this script's source file.
func credentialsFilePath() (string, error) {
	dir, err := scriptDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, "server-credentials"), nil
}

// encKeyFilePath returns the absolute path to the encryption private key file,
// located alongside this script's source file.
func encKeyFilePath() (string, error) {
	dir, err := scriptDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, "enc-key"), nil
}

// domainsFilePath returns the absolute path to the domains JSON file,
// located alongside this script's source file.
func domainsFilePath() (string, error) {
	dir, err := scriptDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, "server-domains.json"), nil
}

