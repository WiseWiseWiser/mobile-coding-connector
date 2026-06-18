## Preconditions

A temporary directory is used as the server config home. The server reads all
configuration from this directory via the `AI_CRITIC_HOME` environment variable,
instead of the default `.ai-critic` directory.

## Steps

1. Create a temporary config home directory
2. Set `AI_CRITIC_HOME` to the temporary directory
3. Write test-specific config files (`opencode.json`, etc.) into the config home
4. Build the server binary (`go build -o <tmp>/ai-critic-server .`) and the basic-auth-proxy binary (`go build -o <tmp>/basic-auth-proxy ./cmd/basic-auth-proxy`)
5. Start the server on a test-controlled port in normal (non-quick-test) mode
6. Wait for the server to become ready (`GET /ping` responds)
7. Capture server stdout/stderr for log verification
8. If `AuthProxyEnabled`: read `basic-auth-proxy.json` to discover the backend port, then check proxy and backend port reachability
9. After the test completes, stop the server process and clean up the temp directory

## Context

The root `Run` function is the test entry point. It accepts a `Request` describing
the desired test scenario and returns a `Response` with collected data for
assertion. The `Request.OpenCodeSettings` controls what is written to
`opencode.json` before the server starts.

This directory tests the server's behaviour when started with various opencode
settings configurations. The key behaviours under test are:

- Whether `AutoStartWebServer()` is triggered (log messages appear)
- Whether the opencode web server starts on the configured port
- Whether the basic-auth-proxy starts and proxies correctly
- How the server handles missing `opencode` binary

### Parameters (ranked by significance)

| # | Parameter | Type | Values | Description |
|---|-----------|------|--------|-------------|
| 1 | `WebServer.Enabled` | bool | true, false | Whether auto-start should trigger |
| 2 | `DefaultDomain` | string | valid domain, "", localhost | Domain for tunnel mapping; localhost skips |
| 3 | `WebServer.Port` | int | 1-65535, 0 (default=4096) | Port for opencode web server |
| 4 | `AuthProxyEnabled` | bool | true, false | Whether basic-auth-proxy wraps the web server |
| 5 | `opencode` binary | availability | present, absent | Whether the opencode CLI is in PATH |
| 6 | `basic-auth-proxy` binary | availability | present, absent | Whether the proxy binary is in PATH (built by test) |
| 7 | `AI_CRITIC_HOME` | env var | path | Custom config home directory |

```go
import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"
	"testing"
	"time"
)

type OpenCodeSettings struct {
	WebServerEnabled bool
	WebServerPort    int
	DefaultDomain    string
	AuthProxyEnabled bool
	BinaryPath       string
}

type PostStartRequest struct {
	URL       string
	Method    string
	Body      string
	Wait      int
	AuthToken string
}

type Request struct {
	OpenCodeSettings *OpenCodeSettings
	ServerPort       int
	TimeoutSecs      int
	PostStart        *PostStartRequest
}

type Response struct {
	ServerPort       int
	ServerStarted    bool
	Logs             string
	PrePostLogs      string
	WebServerRunning bool
	HasAutoStartLog  bool
	BackendPort      int
	BackendRunning   bool
	ProxyRunning     bool
}

func Run(t *testing.T, req *Request) (*Response, error) {
	resp := &Response{}

	if req.ServerPort <= 0 {
		req.ServerPort = 23712
	}
	hash := 0
	for _, c := range t.Name() {
		hash = hash*31 + int(c)
	}
	if hash < 0 {
		hash = -hash
	}
	req.ServerPort = req.ServerPort + (hash % 100)
	if req.TimeoutSecs <= 0 {
		req.TimeoutSecs = 30
	}

	safeName := strings.ReplaceAll(t.Name(), "/", "_")
	safeName = strings.ReplaceAll(safeName, "\\", "_")
	binPath := filepath.Join(os.TempDir(), "ai-critic-server-test-"+safeName)

	buildDir, err := findGoModuleRoot()
	if err != nil {
		return nil, fmt.Errorf("failed to find repo root: %v", err)
	}

	t.Logf("building server binary: %s (dir=%s)", binPath, buildDir)
	build := exec.Command("go", "build", "-o", binPath, ".")
	build.Dir = buildDir
	buildOut, buildErr := build.CombinedOutput()
	if buildErr != nil {
		return nil, fmt.Errorf("failed to build server binary: %v\n%s", buildErr, string(buildOut))
	}
	t.Cleanup(func() {
		os.Remove(binPath)
	})

	proxyBinPath := filepath.Join(os.TempDir(), "basic-auth-proxy-test-"+safeName)
	proxyBuild := exec.Command("go", "build", "-o", proxyBinPath, "./cmd/basic-auth-proxy")
	proxyBuild.Dir = buildDir
	proxyBuildOut, proxyBuildErr := proxyBuild.CombinedOutput()
	if proxyBuildErr != nil {
		return nil, fmt.Errorf("failed to build basic-auth-proxy binary: %v\n%s", proxyBuildErr, string(proxyBuildOut))
	}
	t.Cleanup(func() {
		os.Remove(proxyBinPath)
	})

	binDir := filepath.Dir(binPath)

	serverPort := req.ServerPort
	listener, err := net.Listen("tcp", fmt.Sprintf(":%d", serverPort))
	if err != nil {
		found := false
		for port := serverPort + 1; port < serverPort+100; port++ {
			listener, err = net.Listen("tcp", fmt.Sprintf(":%d", port))
			if err == nil {
				serverPort = port
				found = true
				break
			}
		}
		if !found {
			return nil, fmt.Errorf("no free port found starting from %d", req.ServerPort)
		}
	}
	if serverPort != req.ServerPort {
		t.Logf("port %d was in use, using port %d instead", req.ServerPort, serverPort)
	}
	listener.Close()
	resp.ServerPort = serverPort

	t.Logf("starting server on port %d", serverPort)
	args := []string{"--port", strconv.Itoa(serverPort)}

	configHome := os.Getenv("AI_CRITIC_HOME")
	if configHome == "" {
		var err error
		configHome, err = os.MkdirTemp("", "ai-critic-test-*")
		if err != nil {
			return nil, fmt.Errorf("failed to create temp config home: %v", err)
		}
		os.Setenv("AI_CRITIC_HOME", configHome)
		t.Cleanup(func() {
			os.Unsetenv("AI_CRITIC_HOME")
			os.RemoveAll(configHome)
		})
	}

	testAuthToken := "test-credential-token-for-doctest"
	credFile := filepath.Join(configHome, "server-credentials")
	if err := os.WriteFile(credFile, []byte(testAuthToken+"\n"), 0600); err != nil {
		return nil, fmt.Errorf("failed to write test credentials file: %v", err)
	}
	args = append(args, "--credentials-file", credFile)

	cmd := exec.Command(binPath, args...)
	cmd.Dir = configHome
	env := make([]string, 0, len(os.Environ())+2)
	for _, e := range os.Environ() {
		if strings.HasPrefix(e, "PATH=") {
			env = append(env, "PATH="+binDir+":"+e[5:])
		} else if !strings.HasPrefix(e, "AI_CRITIC_HOME=") {
			env = append(env, e)
		}
	}
	env = append(env, "AI_CRITIC_HOME="+configHome)
	cmd.Env = env

	var logBuf bytes.Buffer
	cmd.Stdout = &logBuf
	cmd.Stderr = &logBuf

	if err := cmd.Start(); err != nil {
		return nil, fmt.Errorf("failed to start server: %v", err)
	}

	wsCleanupPort := 0
	if req.OpenCodeSettings != nil {
		wsCleanupPort = req.OpenCodeSettings.WebServerPort
	}
	if wsCleanupPort <= 0 {
		wsCleanupPort = 14096
	}

	stopServer := func() {
		if cmd.Process != nil {
			cmd.Process.Signal(os.Interrupt)
			time.Sleep(500 * time.Millisecond)
			cmd.Process.Signal(syscall.SIGTERM)
			time.Sleep(200 * time.Millisecond)
			cmd.Process.Kill()
		}
		killPort(wsCleanupPort)
	}
	t.Cleanup(stopServer)

	pingURL := fmt.Sprintf("http://localhost:%d/ping", serverPort)
	deadline := time.Now().Add(time.Duration(req.TimeoutSecs) * time.Second)

	var started bool
	for time.Now().Before(deadline) {
		if httpGetOK(pingURL) {
			started = true
			break
		}
		time.Sleep(500 * time.Millisecond)
	}

	resp.ServerStarted = started

	if started {
		time.Sleep(3 * time.Second)

		if req.PostStart != nil {
			resp.PrePostLogs = logBuf.String()

			if testAuthToken != "" {
				req.PostStart.AuthToken = testAuthToken
			}

			url := strings.ReplaceAll(req.PostStart.URL, "__PORT__", strconv.Itoa(serverPort))
			t.Logf("PostStart: making %s request to %s", req.PostStart.Method, url)
			if err := postStartHTTP(req.PostStart.Method, url, req.PostStart.Body, req.PostStart.AuthToken); err != nil {
				t.Logf("PostStart: request failed: %v", err)
			} else {
				t.Logf("PostStart: request completed successfully")
			}

			if wait := req.PostStart.Wait; wait > 0 {
				t.Logf("PostStart: waiting %d seconds for effects...", wait)
				time.Sleep(time.Duration(wait) * time.Second)
			}
		}

		settings := req.OpenCodeSettings
		if settings != nil {
			wsPort := settings.WebServerPort
			if wsPort <= 0 {
				wsPort = 14096
			}

			if settings.AuthProxyEnabled {
				resp.ProxyRunning = checkPort(wsPort)

				proxyConfigPath := filepath.Join(configHome, "basic-auth-proxy.json")
				proxyData, readErr := os.ReadFile(proxyConfigPath)
				if readErr == nil {
					var proxyCfg struct {
						BackendPort int `json:"backend_port"`
					}
					if jsonErr := json.Unmarshal(proxyData, &proxyCfg); jsonErr == nil && proxyCfg.BackendPort > 0 {
						resp.BackendPort = proxyCfg.BackendPort
						resp.BackendRunning = checkPort(proxyCfg.BackendPort)
					}
				}
			} else {
				resp.WebServerRunning = checkPort(wsPort)
			}
		}
	}

	stopServer()
	_ = cmd.Wait()

	resp.Logs = logBuf.String()
	resp.HasAutoStartLog = strings.Contains(resp.Logs, "[opencode] AutoStartWebServer:")

	t.Logf("=== SERVER OUTPUT ===\n%s\n=== END SERVER OUTPUT ===", resp.Logs)
	t.Logf("HasAutoStartLog: %v, LogsLen: %d", resp.HasAutoStartLog, len(resp.Logs))

	return resp, nil
}

func findGoModuleRoot() (string, error) {
	dir, err := os.Getwd()
	if err != nil {
		return "", err
	}
	for {
		if _, err := os.Stat(filepath.Join(dir, "go.mod")); err == nil {
			return dir, nil
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			return "", fmt.Errorf("go.mod not found in any parent of %s", dir)
		}
		dir = parent
	}
}

func isPortInUse(port int) bool {
	conn, err := net.DialTimeout("tcp", fmt.Sprintf("localhost:%d", port), 500*time.Millisecond)
	if err != nil {
		return false
	}
	conn.Close()
	return true
}

func httpGetOK(url string) bool {
	resp, err := http.Get(url)
	if err != nil {
		return false
	}
	resp.Body.Close()
	return resp.StatusCode == http.StatusOK
}

func checkPort(port int) bool {
	conn, err := net.DialTimeout("tcp", fmt.Sprintf("localhost:%d", port), 1*time.Second)
	if err != nil {
		return false
	}
	conn.Close()
	return true
}

func killPort(port int) {
	out, err := exec.Command("lsof", "-ti", fmt.Sprintf("tcp:%d", port)).Output()
	if err != nil {
		return
	}
	pids := strings.Fields(strings.TrimSpace(string(out)))
	for _, pidStr := range pids {
		pid, err := strconv.Atoi(pidStr)
		if err != nil {
			continue
		}
		if err := syscall.Kill(pid, syscall.SIGTERM); err != nil {
			_ = syscall.Kill(pid, syscall.SIGKILL)
		}
	}
}

func postStartHTTP(method, url, body string, authToken string) error {
	req, err := http.NewRequest(method, url, strings.NewReader(body))
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	if authToken != "" {
		req.Header.Set("Authorization", "Bearer "+authToken)
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		b, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("unexpected status %d: %s", resp.StatusCode, string(b))
	}
	return nil
}
```
