package lib

import (
	"context"
	"crypto/sha256"
	"fmt"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"strings"
	"syscall"
	"time"

	"github.com/xhd2015/lifelog-private/ai-critic/server/config"
)

const quickTestConfigLabel = "ai-critic.config-hash"

type SandboxQuickTestOptions struct {
	ArchFlag      string
	ContainerPort int
	ContainerName string
	ScriptSubDir  string
}

type SandboxQuickTestResult struct {
	ServerCmd     *exec.Cmd
	ViteCmd       *exec.Cmd
	ContainerName string
}

// SandboxQuickTestPrepare builds the server, sets up the container, and
// copies the binary. Call SandboxQuickTestStart afterwards to start vite and
// the server inside the container.
func SandboxQuickTestPrepare(opts SandboxQuickTestOptions) error {
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
				"  Use --arch %s or --arch auto for local testing.",
			goarch, vmArch, vmArch,
		)
	}

	binaryPath := fmt.Sprintf("/tmp/ai-critic-linux-%s", goarch)
	fmt.Printf("\n=== Step 1: Cross-compiling Go server for linux/%s ===\n", goarch)
	if err := BuildServer(BuildServerOptions{
		Output: binaryPath,
		GOOS:   "linux",
		GOARCH: goarch,
	}); err != nil {
		return err
	}

	fmt.Println("\n=== Step 2: Setting up podman container ===")
	files, err := setupSandboxFiles(opts.ScriptSubDir)
	if err != nil {
		return err
	}

	containerName := opts.ContainerName
	containerPort := opts.ContainerPort

	status, _ := InspectContainerStatus(containerName)
	if status != "running" && CheckPort(containerPort) {
		fmt.Printf("Port %d is in use, killing existing process...\n", containerPort)
		killedPid, killErr := KillPortPid(containerPort)
		if killErr != nil {
			return fmt.Errorf("failed to kill process on port %d: %v", containerPort, killErr)
		}
		if killedPid > 0 {
			fmt.Printf("Killed process (PID: %d), waiting for port release...\n", killedPid)
			time.Sleep(500 * time.Millisecond)
		}
	}

	if err := ensureQuickTestContainer(containerName, goarch, containerPort, files); err != nil {
		return err
	}

	fmt.Println("Checking for existing server process in container...")
	_ = exec.Command("podman", "exec", containerName, "pkill", "-f", "ai-critic").Run()

	fmt.Println("Copying binary into container...")
	if err := RunVerbose("podman", "cp", binaryPath, containerName+":/usr/local/bin/ai-critic"); err != nil {
		return fmt.Errorf("failed to copy binary: %v", err)
	}

	return nil
}

// SandboxQuickTestStart starts the vite dev server on the host (if needed)
// and execs the server binary inside the container. The exec cmd is started
// but NOT waited on — the caller manages the lifecycle via the returned result.
func SandboxQuickTestStart(ctx context.Context, opts SandboxQuickTestOptions) (*SandboxQuickTestResult, error) {
	containerName := opts.ContainerName
	containerPort := opts.ContainerPort

	projectDir := resolveDefaultProjectDir()
	var viteCmd *exec.Cmd
	if !CheckPort(ViteDevPort) {
		fmt.Println("\n=== Starting Vite dev server ===")
		var err error
		viteCmd, err = startViteDevServer(projectDir)
		if err != nil {
			return nil, err
		}
	} else {
		fmt.Printf("Vite dev server already running on port %d\n", ViteDevPort)
	}

	serverArgs := []string{
		"exec", containerName,
		"/usr/local/bin/ai-critic",
		"--quick-test",
		fmt.Sprintf("--port=%d", containerPort),
		"--dev",
		"--frontend-port", fmt.Sprintf("%d", ViteDevPort),
		"--frontend-host", "host.containers.internal",
		"--credentials-file", "/root/" + config.CredentialsFile,
		"--enc-key-file", "/root/" + config.EncKeyFile,
		"--domains-file", "/root/" + config.DomainsFile,
	}

	fmt.Printf("\nServer: http://localhost:%d\n", containerPort)
	fmt.Printf("Frontend: http://localhost:%d (proxied from host)\n", ViteDevPort)

	execCmd := exec.CommandContext(ctx, "podman", serverArgs...)
	execCmd.Stdout = os.Stdout
	execCmd.Stderr = os.Stderr
	execCmd.Stdin = os.Stdin

	if err := execCmd.Start(); err != nil {
		if viteCmd != nil && viteCmd.Process != nil {
			viteCmd.Process.Kill()
		}
		return nil, fmt.Errorf("failed to start server in container: %v", err)
	}

	return &SandboxQuickTestResult{
		ServerCmd:     execCmd,
		ViteCmd:       viteCmd,
		ContainerName: containerName,
	}, nil
}

// RunSandboxQuickTest runs the server in quick-test mode inside a container
// using podman exec. The container stays alive (sleep infinity) across runs.
// Vite runs on the host and the server proxies frontend requests to it.
// This is a blocking convenience wrapper around SandboxQuickTestPrepare + SandboxQuickTestStart.
func RunSandboxQuickTest(opts SandboxQuickTestOptions) error {
	if err := SandboxQuickTestPrepare(opts); err != nil {
		return err
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)
	interrupted := false
	go func() {
		<-sigChan
		fmt.Println("\nReceived interrupt, shutting down...")
		interrupted = true
		cancel()
	}()

	result, err := SandboxQuickTestStart(ctx, opts)
	if err != nil {
		return err
	}

	fmt.Println("Press Ctrl+C to stop.")
	fmt.Println()

	serverErr := result.ServerCmd.Wait()

	containerName := result.ContainerName
	if !interrupted {
		time.Sleep(2 * time.Second)
		if exec.Command("podman", "exec", containerName, "pgrep", "-f", "ai-critic").Run() == nil {
			fmt.Println("New server instance detected, skipping cleanup.")
			return nil
		}
	}

	fmt.Println("\nCleaning up...")
	fmt.Println("Stopping container...")
	_ = exec.Command("podman", "stop", containerName).Run()

	if result.ViteCmd != nil && result.ViteCmd.Process != nil {
		fmt.Println("Stopping Vite dev server...")
		result.ViteCmd.Process.Signal(syscall.SIGTERM)
		result.ViteCmd.Wait()
	}

	if serverErr != nil && ctx.Err() == nil {
		return fmt.Errorf("server exited with error: %v", serverErr)
	}
	return nil
}

// quickTestContainerConfig returns the config-relevant portion used
// to detect when the container needs recreation.
func quickTestContainerConfig(goarch string, containerPort int, files *sandboxFiles) string {
	return strings.Join([]string{
		"platform=" + fmt.Sprintf("linux/%s", goarch),
		"home=" + files.homeDir,
		"apt-archives=" + files.aptArchivesDir,
		"apt-lists=" + files.aptListsDir,
		"downloads=" + files.downloadsDir,
		"data=" + files.dataDir,
		"port=" + fmt.Sprintf("%d", containerPort),
		"image=" + ContainerImage,
	}, "\n")
}

func configHash(cfg string) string {
	h := sha256.Sum256([]byte(cfg))
	return fmt.Sprintf("%x", h[:8])
}

func inspectContainerLabel(containerName, label string) string {
	var buf strings.Builder
	c := exec.Command("podman", "inspect", "--format", fmt.Sprintf("{{index .Config.Labels %q}}", label), containerName)
	c.Stdout = &buf
	if err := c.Run(); err != nil {
		return ""
	}
	return strings.TrimSpace(buf.String())
}

func ensureQuickTestContainer(containerName, goarch string, containerPort int, files *sandboxFiles) error {
	cfg := quickTestContainerConfig(goarch, containerPort, files)
	wantHash := configHash(cfg)

	status, err := InspectContainerStatus(containerName)
	if err != nil {
		return createAndStartQuickTestContainer(containerName, goarch, containerPort, files, wantHash)
	}

	if gotHash := inspectContainerLabel(containerName, quickTestConfigLabel); gotHash != wantHash {
		fmt.Printf("Container %q config changed (got %s, want %s), recreating...\n", containerName, gotHash, wantHash)
		_ = RunVerbose("podman", "rm", "-f", containerName)
		return createAndStartQuickTestContainer(containerName, goarch, containerPort, files, wantHash)
	}

	if status == "running" {
		fmt.Printf("Reusing running container %q\n", containerName)
		return nil
	}

	fmt.Printf("Container %q is stopped, restarting...\n", containerName)
	if err := RunVerbose("podman", "start", containerName); err != nil {
		return fmt.Errorf("failed to start container: %v", err)
	}
	return nil
}

func createAndStartQuickTestContainer(containerName, goarch string, containerPort int, files *sandboxFiles, configHash string) error {
	platform := fmt.Sprintf("linux/%s", goarch)
	fmt.Printf("Creating container (platform: %s)...\n", platform)

	args := []string{
		"create",
		"--name", containerName,
		"--platform", platform,
		"-w", "/root",
		"-v", files.homeDir + ":/root",
		"-v", files.aptArchivesDir + ":/var/cache/apt/archives",
		"-v", files.aptListsDir + ":/var/lib/apt/lists",
		"-v", files.downloadsDir + ":/tmp/downloads",
		"-v", files.dataDir + ":/root/" + config.DataDir,
		"-p", fmt.Sprintf("%d:%d", containerPort, containerPort),
		"--add-host=host.containers.internal:host-gateway",
		"--label", quickTestConfigLabel + "=" + configHash,
		ContainerImage,
		"sleep", "infinity",
	}

	if err := RunVerbose("podman", args...); err != nil {
		return fmt.Errorf("failed to create container: %v", err)
	}

	fmt.Println("Starting container...")
	if err := RunVerbose("podman", "start", containerName); err != nil {
		return fmt.Errorf("failed to start container: %v", err)
	}
	return nil
}

func startViteDevServer(projectDir string) (*exec.Cmd, error) {
	// Use --host to bind on 0.0.0.0 so the container can reach Vite via host.containers.internal
	viteCmd := exec.Command("npm", "run", "dev", "--", "--host")
	viteCmd.Dir = filepath.Join(projectDir, "ai-critic-react")
	viteCmd.Stdout = os.Stdout
	viteCmd.Stderr = os.Stderr

	if err := viteCmd.Start(); err != nil {
		return nil, fmt.Errorf("failed to start vite: %v", err)
	}

	fmt.Printf("Waiting for Vite dev server on port %d...\n", ViteDevPort)
	if err := waitForHTTP(context.Background(), fmt.Sprintf("http://localhost:%d", ViteDevPort), 30*time.Second); err != nil {
		viteCmd.Process.Kill()
		return nil, fmt.Errorf("vite failed to start: %v", err)
	}
	fmt.Println("Vite dev server is ready!")
	return viteCmd, nil
}
