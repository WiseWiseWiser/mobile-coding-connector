package lib

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"time"

	"github.com/xhd2015/xgo/support/cmd"
)

type QuickTestOptions struct {
	Port         int  // Server port (default: QuickTestPort)
	NoVite       bool // If true, don't start vite and use static frontend
	FrontendPort int  // If > 0, proxy to this port (default: ViteDevPort if !NoVite)
	Keep         bool // If true, add --keep flag
	ProjectDir   string

	Stdout io.Writer
	Stderr io.Writer
	Stdin  io.Reader

	restarted bool
}

type QuickTestResult struct {
	ServerCmd *exec.Cmd
	ViteCmd   *exec.Cmd
	Restarted bool
}

func (o *QuickTestOptions) GetPort() int {
	if o.Port == 0 {
		return QuickTestPort
	}
	return o.Port
}

func (o *QuickTestOptions) GetProjectDir() string {
	if o.ProjectDir != "" {
		return o.ProjectDir
	}
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return ""
	}
	return filepath.Join(homeDir, "mobile-coding-connector")
}

func (o *QuickTestOptions) GetFrontendPort() int {
	if o.FrontendPort > 0 {
		return o.FrontendPort
	}
	if !o.NoVite {
		return ViteDevPort
	}
	return 0
}

func QuickTestPrepare(opts *QuickTestOptions) error {
	port := opts.GetPort()
	projectDir := opts.GetProjectDir()

	fmt.Println("Building Go server...")
	if err := QuickTestBuild(projectDir); err != nil {
		return err
	}

	binaryPath := "/tmp/ai-critic-quick"

	if CheckPort(port) {
		fmt.Printf("Port %d is in use, trying exec-restart...\n", port)
		if tryExecRestart(port, binaryPath) {
			fmt.Println("Server restarted via exec (PID preserved)")
			opts.restarted = true
			return nil
		}
		return fmt.Errorf("port %d is in use but exec-restart failed - ensure quick-test server is running", port)
	}

	if !opts.NoVite {
		fmt.Printf("Checking for existing vite on port %d...\n", ViteDevPort)
		killedVitePid, err := KillPortPid(ViteDevPort)
		if err != nil {
			return err
		}
		if killedVitePid > 0 {
			fmt.Printf("Killed previous vite (PID: %d)\n", killedVitePid)
		}
	}

	return nil
}

func tryExecRestart(port int, binaryPath string) bool {
	url := fmt.Sprintf("http://localhost:%d/api/quick-test/exec-restart?binary=%s", port, binaryPath)

	client := &http.Client{Timeout: 3 * time.Second}
	resp, err := client.Post(url, "application/json", nil)
	if err != nil {
		return false
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return false
	}

	var result struct {
		Status  string `json:"status"`
		Message string `json:"message"`
		Binary  string `json:"binary"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return false
	}

	if result.Status != "restarting" {
		return false
	}

	fmt.Printf("Waiting for server to restart...\n")
	deadline := time.Now().Add(5 * time.Second)
	for time.Now().Before(deadline) {
		resp, err := client.Get(fmt.Sprintf("http://localhost:%d/ping", port))
		if err == nil {
			resp.Body.Close()
			return true
		}
		time.Sleep(100 * time.Millisecond)
	}

	return false
}

func QuickTestBuild(projectDir string) error {
	if projectDir == "" {
		homeDir, err := os.UserHomeDir()
		if err != nil {
			return fmt.Errorf("failed to get home directory: %v", err)
		}
		projectDir = filepath.Join(homeDir, "mobile-coding-connector")
	}

	return cmd.Debug().Dir(projectDir).Run("go", "build", "-o", "/tmp/ai-critic-quick", "./")
}

func QuickTestStart(ctx context.Context, opts *QuickTestOptions) (*QuickTestResult, error) {
	if opts.restarted {
		return &QuickTestResult{Restarted: true}, nil
	}

	port := opts.GetPort()
	projectDir := opts.GetProjectDir()
	frontendPort := opts.GetFrontendPort()

	homeDir, err := os.UserHomeDir()
	if err != nil {
		return nil, fmt.Errorf("failed to get home directory: %v", err)
	}

	var viteCmd *exec.Cmd
	if !opts.NoVite && frontendPort > 0 {
		fmt.Println("Starting Vite dev server...")
		viteCmd = exec.CommandContext(ctx, "npm", "run", "dev")
		viteCmd.Dir = filepath.Join(projectDir, "ai-critic-react")
		if opts.Stdout != nil {
			viteCmd.Stdout = opts.Stdout
		} else {
			viteCmd.Stdout = os.Stdout
		}
		if opts.Stderr != nil {
			viteCmd.Stderr = opts.Stderr
		} else {
			viteCmd.Stderr = os.Stderr
		}

		if err := viteCmd.Start(); err != nil {
			return nil, fmt.Errorf("failed to start vite: %v", err)
		}

		fmt.Printf("Waiting for Vite dev server on port %d...\n", frontendPort)
		if err := waitForHTTP(ctx, fmt.Sprintf("http://localhost:%d", frontendPort), 30*time.Second); err != nil {
			if viteCmd.Process != nil {
				viteCmd.Process.Kill()
			}
			return nil, fmt.Errorf("vite failed to start: %v", err)
		}
		fmt.Println("Vite dev server is ready!")
	}

	args := []string{"--quick-test", fmt.Sprintf("--port=%d", port)}
	if projectDir != "" {
		args = append(args, "--project-dir", projectDir)
	}
	if frontendPort > 0 {
		args = append(args, "--dev", "--frontend-port", fmt.Sprintf("%d", frontendPort))
	}
	if opts.Keep {
		args = append(args, "--keep")
	}

	fmt.Printf("Executing: /tmp/ai-critic-quick %s\n", argsToString(args))

	serverCmd := exec.Command("/tmp/ai-critic-quick", args...)
	serverCmd.Dir = homeDir

	if opts.Stdout != nil {
		serverCmd.Stdout = opts.Stdout
	} else {
		serverCmd.Stdout = os.Stdout
	}
	if opts.Stderr != nil {
		serverCmd.Stderr = opts.Stderr
	} else {
		serverCmd.Stderr = os.Stderr
	}
	if opts.Stdin != nil {
		serverCmd.Stdin = opts.Stdin
	} else {
		serverCmd.Stdin = os.Stdin
	}

	if err := serverCmd.Start(); err != nil {
		if viteCmd != nil && viteCmd.Process != nil {
			viteCmd.Process.Kill()
		}
		return nil, fmt.Errorf("failed to start server: %v", err)
	}

	return &QuickTestResult{
		ServerCmd: serverCmd,
		ViteCmd:   viteCmd,
	}, nil
}

func QuickTestCommand(opts *QuickTestOptions) (*exec.Cmd, error) {
	port := opts.GetPort()
	projectDir := opts.GetProjectDir()
	frontendPort := opts.GetFrontendPort()

	homeDir, err := os.UserHomeDir()
	if err != nil {
		return nil, fmt.Errorf("failed to get home directory: %v", err)
	}

	args := []string{"--quick-test", fmt.Sprintf("--port=%d", port)}
	if projectDir != "" {
		args = append(args, "--project-dir", projectDir)
	}
	if frontendPort > 0 {
		args = append(args, "--dev", "--frontend-port", fmt.Sprintf("%d", frontendPort))
	}
	if opts.Keep {
		args = append(args, "--keep")
	}

	fmt.Printf("Executing: /tmp/ai-critic-quick %s\n", argsToString(args))

	serverCmd := exec.Command("/tmp/ai-critic-quick", args...)
	serverCmd.Dir = homeDir

	if opts.Stdout != nil {
		serverCmd.Stdout = opts.Stdout
	} else {
		serverCmd.Stdout = os.Stdout
	}
	if opts.Stderr != nil {
		serverCmd.Stderr = opts.Stderr
	} else {
		serverCmd.Stderr = os.Stderr
	}
	if opts.Stdin != nil {
		serverCmd.Stdin = opts.Stdin
	} else {
		serverCmd.Stdin = os.Stdin
	}

	return serverCmd, nil
}

func waitForHTTP(ctx context.Context, url string, timeout time.Duration) error {
	deadline := time.Now().Add(timeout)
	client := &http.Client{Timeout: 1 * time.Second}
	for time.Now().Before(deadline) {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}
		resp, err := client.Get(url)
		if err == nil {
			resp.Body.Close()
			return nil
		}
		time.Sleep(200 * time.Millisecond)
	}
	return fmt.Errorf("timeout waiting for HTTP at %s", url)
}

func argsToString(args []string) string {
	result := ""
	for i, arg := range args {
		if i > 0 {
			result += " "
		}
		if containsSpace(arg) {
			result += fmt.Sprintf("%q", arg)
		} else {
			result += arg
		}
	}
	return result
}

func containsSpace(s string) bool {
	for _, c := range s {
		if c == ' ' {
			return true
		}
	}
	return false
}
