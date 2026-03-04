package lib

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"strings"
	"syscall"

	"github.com/xhd2015/lifelog-private/ai-critic/server/config"
	"github.com/xhd2015/xgo/support/cmd"
)

type SandboxOptions struct {
	ArchFlag      string // "auto", "amd64", "arm64"
	ScriptSubDir  string // relative path under repo root for config files (e.g. "script/sandbox/fresh-setup")
	FreshSetup    bool   // true = always destroy and recreate container
	ContainerPort int
	ContainerName string // podman container name
}

// RunSandbox builds the frontend and server, then runs them in a podman container.
func RunSandbox(opts SandboxOptions) error {
	if err := EnsurePodman(); err != nil {
		return err
	}

	goarch, err := ResolveArch(opts.ArchFlag)
	if err != nil {
		return err
	}

	vmArch, vmErr := PodmanArch()
	if vmErr == nil && vmArch != goarch {
		return fmt.Errorf(
			"target arch %q differs from podman VM arch %q.\n"+
				"  Go binaries crash under Rosetta/QEMU emulation (SIGSEGV in netpoll_epoll).\n"+
				"  Use --arch %s or --arch auto for local testing.\n"+
				"  For amd64 builds, use a real amd64 machine or CI/CD.",
			goarch, vmArch, vmArch,
		)
	}

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

	binaryPath := fmt.Sprintf("/tmp/ai-critic-linux-%s", goarch)
	fmt.Printf("\n=== Step 2: Cross-compiling Go server for linux/%s ===\n", goarch)
	if err := BuildServer(BuildServerOptions{
		Output: binaryPath,
		GOOS:   "linux",
		GOARCH: goarch,
	}); err != nil {
		return err
	}

	fmt.Println("\n=== Step 3: Setting up podman container ===")
	sandboxFiles, err := setupSandboxFiles(opts.ScriptSubDir)
	if err != nil {
		return err
	}

	name := opts.ContainerName
	containerPort := opts.ContainerPort
	if opts.FreshSetup {
		return runFreshContainer(name, goarch, binaryPath, containerPort, sandboxFiles)
	}
	return runBootContainer(name, goarch, binaryPath, containerPort, sandboxFiles)
}

// ResolveArch resolves the target architecture from an --arch flag value.
// "auto" detects the podman VM's architecture, otherwise validates the specified value.
func ResolveArch(archFlag string) (string, error) {
	if archFlag == "auto" {
		return PodmanArch()
	}
	switch archFlag {
	case "amd64", "arm64":
		return archFlag, nil
	default:
		return "", fmt.Errorf("unsupported architecture: %s (supported: auto, amd64, arm64)", archFlag)
	}
}

type sandboxFiles struct {
	aptArchivesDir string
	aptListsDir    string
	downloadsDir   string
	dataDir        string // host-side .ai-critic directory, mounted as /root/.ai-critic
	homeDir        string // host-side home directory, mounted as /root to persist across restarts
}

func setupSandboxFiles(scriptSubDir string) (*sandboxFiles, error) {
	systemCacheDir, err := os.UserCacheDir()
	if err != nil {
		return nil, fmt.Errorf("failed to get system cache directory: %v", err)
	}
	cacheBase := systemCacheDir + "/ai-critic"
	files := &sandboxFiles{
		aptArchivesDir: cacheBase + "/apt-archives",
		aptListsDir:    cacheBase + "/apt-lists",
		downloadsDir:   cacheBase + "/downloads",
	}
	for _, dir := range []string{files.aptArchivesDir, files.aptListsDir, files.downloadsDir} {
		if err := os.MkdirAll(dir, 0755); err != nil {
			return nil, fmt.Errorf("failed to create cache dir %s: %v", dir, err)
		}
	}
	fmt.Printf("Cache directory: %s\n", cacheBase)

	baseDir, err := repoSubDir(scriptSubDir)
	if err != nil {
		return nil, err
	}

	files.homeDir = filepath.Join(baseDir, "home")
	if err := os.MkdirAll(files.homeDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create home dir %s: %v", files.homeDir, err)
	}
	fmt.Printf("Home directory: %s\n", files.homeDir)

	files.dataDir = filepath.Join(baseDir, config.DataDir)
	if err := os.MkdirAll(files.dataDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create data dir %s: %v", files.dataDir, err)
	}
	fmt.Printf("Data directory: %s\n", files.dataDir)

	// Ensure essential config files exist (empty if missing)
	essentialFiles := []struct {
		path string
		perm os.FileMode
	}{
		{filepath.Join(baseDir, config.CredentialsFile), 0600},
		{filepath.Join(baseDir, config.EncKeyFile), 0600},
		{filepath.Join(baseDir, config.EncKeyPubFile), 0600},
		{filepath.Join(baseDir, config.DomainsFile), 0644},
	}
	for _, f := range essentialFiles {
		if err := ensureFileExists(f.path, f.perm); err != nil {
			return nil, fmt.Errorf("failed to create config file %s: %v", f.path, err)
		}
	}

	return files, nil
}

func containerCreateArgs(containerName, goarch string, containerPort int, files *sandboxFiles, mountWholeDataDir bool) []string {
	containerCredentialsFile := "/root/" + config.CredentialsFile
	containerEncKeyFile := "/root/" + config.EncKeyFile
	containerDomainsFile := "/root/" + config.DomainsFile
	platform := fmt.Sprintf("linux/%s", goarch)

	args := []string{
		"create",
		"--name", containerName,
		"--platform", platform,
		"-w", "/root",
		"-v", files.homeDir + ":/root",
		"-v", files.aptArchivesDir + ":/var/cache/apt/archives",
		"-v", files.aptListsDir + ":/var/lib/apt/lists",
		"-v", files.downloadsDir + ":/tmp/downloads",
	}

	if mountWholeDataDir {
		args = append(args, "-v", files.dataDir+":/root/"+config.DataDir)
	} else {
		args = append(args,
			"-v", filepath.Join(files.dataDir, filepath.Base(config.CredentialsFile))+":"+containerCredentialsFile,
			"-v", filepath.Join(files.dataDir, filepath.Base(config.EncKeyFile))+":"+"/root/"+config.EncKeyFile,
			"-v", filepath.Join(files.dataDir, filepath.Base(config.EncKeyPubFile))+":"+"/root/"+config.EncKeyPubFile,
			"-v", filepath.Join(files.dataDir, filepath.Base(config.DomainsFile))+":"+containerDomainsFile,
		)
	}

	args = append(args,
		"-p", fmt.Sprintf("%d:%d", containerPort, containerPort),
		ContainerImage,
		"/usr/local/bin/ai-critic", "--port", fmt.Sprintf("%d", containerPort),
		"--credentials-file", containerCredentialsFile,
		"--enc-key-file", containerEncKeyFile,
		"--domains-file", containerDomainsFile,
	)
	return args
}

func runFreshContainer(containerName, goarch, binaryPath string, containerPort int, files *sandboxFiles) error {
	fmt.Println("Removing old container (if any)...")
	_ = RunVerbose("podman", "rm", "-f", containerName)

	platform := fmt.Sprintf("linux/%s", goarch)
	fmt.Printf("Creating container (platform: %s)...\n", platform)
	if err := RunVerbose("podman", containerCreateArgs(containerName, goarch, containerPort, files, false)...); err != nil {
		return fmt.Errorf("failed to create container: %v", err)
	}

	fmt.Println("Copying binary into container...")
	if err := RunVerbose("podman", "cp", binaryPath, containerName+":/usr/local/bin/ai-critic"); err != nil {
		return fmt.Errorf("failed to copy binary into container: %v", err)
	}

	fmt.Printf("\nStarting container (platform: %s)...\nServer will be available at http://localhost:%d\n\n", platform, containerPort)
	if err := RunVerbose("podman", "start", containerName); err != nil {
		return fmt.Errorf("failed to start container: %v", err)
	}

	return followContainerLogs(containerName)
}

func runBootContainer(containerName, goarch, binaryPath string, containerPort int, files *sandboxFiles) error {
	needsCreate := false
	status, err := InspectContainerStatus(containerName)
	if err != nil {
		needsCreate = true
	} else if !inspectContainerHasPort(containerName, containerPort) {
		fmt.Printf("Container %q port mapping mismatch (want %d). Recreating...\n", containerName, containerPort)
		_ = RunVerbose("podman", "rm", "-f", containerName)
		needsCreate = true
	} else if status == "running" {
		fmt.Printf("Stopping running container %q...\n", containerName)
		_ = RunVerbose("podman", "stop", containerName)
	} else {
		fmt.Printf("Reusing existing container %q (status: %s)\n", containerName, status)
	}

	if needsCreate {
		platform := fmt.Sprintf("linux/%s", goarch)
		fmt.Printf("Creating container (platform: %s)...\n", platform)
		if err := RunVerbose("podman", containerCreateArgs(containerName, goarch, containerPort, files, true)...); err != nil {
			return fmt.Errorf("failed to create container: %v", err)
		}
	}

	fmt.Println("Copying binary into container...")
	if err := RunVerbose("podman", "cp", binaryPath, containerName+":/usr/local/bin/ai-critic"); err != nil {
		return fmt.Errorf("failed to copy binary into container: %v", err)
	}

	fmt.Printf("\nStarting container...\nServer will be available at http://localhost:%d\n\n", containerPort)
	if err := RunVerbose("podman", "start", containerName); err != nil {
		return fmt.Errorf("failed to start container: %v", err)
	}

	return followContainerLogs(containerName)
}

// inspectContainerHasPort checks whether the container's port bindings
// include the desired port. Returns false if the container doesn't exist
// or the port is not mapped.
func inspectContainerHasPort(containerName string, containerPort int) bool {
	var buf bytes.Buffer
	c := exec.Command("podman", "inspect", "--format", "{{json .HostConfig.PortBindings}}", containerName)
	c.Stdout = &buf
	c.Stderr = &buf
	if err := c.Run(); err != nil {
		return false
	}
	return strings.Contains(buf.String(), fmt.Sprintf("%d/tcp", containerPort))
}

// InspectContainerStatus returns the podman status of the named container,
// or an error if the container does not exist.
func InspectContainerStatus(containerName string) (string, error) {
	var buf bytes.Buffer
	c := exec.Command("podman", "inspect", "--format", "{{.State.Status}}", containerName)
	c.Stdout = &buf
	c.Stderr = &buf
	if err := c.Run(); err != nil {
		return "", err
	}
	return strings.TrimSpace(buf.String()), nil
}

func followContainerLogs(containerName string) error {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)
	go func() {
		<-sigChan
		fmt.Println("\nShutting down container...")
		cancel()
	}()

	logsCmd := exec.CommandContext(ctx, "podman", "logs", "-f", containerName)
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
		_ = exec.Command("podman", "stop", containerName).Run()
	case err := <-done:
		if err != nil && ctx.Err() == nil {
			return fmt.Errorf("container exited with error: %v", err)
		}
	}

	return nil
}

func repoSubDir(subDir string) (string, error) {
	out, err := exec.Command("git", "rev-parse", "--show-toplevel").Output()
	if err != nil {
		return "", fmt.Errorf("failed to get git repo root: %v", err)
	}
	repoRoot := strings.TrimSpace(string(out))
	return filepath.Join(repoRoot, subDir), nil
}

func ensureFileExists(path string, perm os.FileMode) error {
	if _, err := os.Stat(path); os.IsNotExist(err) {
		return os.WriteFile(path, nil, perm)
	}
	return nil
}
