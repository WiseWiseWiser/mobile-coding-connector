# Streaming Integration Doctests

End-to-end tests that start `ai-critic-server`, run the real `remote-agent`
CLI, and verify incremental doctor output over the full HTTP → SSE → streamcmd
path.

# DSN (Domain Specific Notion)

The streaming integration harness models the production doctor command path
without relying on mock SSE alone.

**Participants**

- **ai-critic-server** — serves `GET /api/ws-proxy/doctor/stream`; emits
  server checks as SSE `progress` frames.
- **remote-agent subprocess** — `ws-proxy doctor` via `streamcmd.Run`; stdout
  read line-by-line through a pipe.
- **Test config home** — isolated `AI_CRITIC_HOME` with credentials and seeded
  `ws-proxy.json`.
- **wsproxy test hooks** — stub external network checks; optional delay on
  `upstream_fetch` for incremental timing proofs.

**Behaviors**

- Server check lines (`[ok]`/`[fail]`/`[skip]`) appear on stdout before the
  process exits.
- Slow checks prove interleaved output (configuration load before upstream fetch).
- Exit code is 0 when healthy, non-zero when failing checks remain.
- Client-side checks run after server stream via `After` hook.

## Version

0.0.2

## Decision Tree

```
[streaming integration]
 |
 +-- doctor/
      |
      +-- streams-from-server/                 (LEAF)  stdout has checks before exit
      +-- prints-incrementally/                (LEAF)  config line ≥150ms before fetch line
      +-- exit-code-reflects-health/            (LEAF)  non-zero exit when unhealthy
```

## Test Index

| # | Leaf | Description |
|---|------|-------------|
| 1 | `doctor/streams-from-server` | Doctor header, check lines, Result line before exit |
| 2 | `doctor/prints-incrementally` | Early check line precedes delayed upstream fetch line |
| 3 | `doctor/exit-code-reflects-health` | Exit code 1 when doctor finds failing checks |

## How to Run

```sh
doctest vet ./tests/streaming
doctest test ./tests/streaming/...
```

```go
import (
	"bufio"
	"bytes"
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

	"github.com/xhd2015/ai-critic/script/lib"
	"github.com/xhd2015/ai-critic/server/proxy/wsproxy"
)

type Request struct {
	UpstreamFetchDelayMs int
	RecordLineTimes      bool
	ExpectExitZero       *bool
}

type LineRecord struct {
	Line string
	At   time.Time
}

type Response struct {
	ServerPort    int
	ExitCode      int
	Stdout        string
	StdoutLines   []string
	LineRecords   []LineRecord
	HasDoctorHdr  bool
	HasCheckLines bool
	HasResultLine bool
}

func Run(t *testing.T, req *Request) (*Response, error) {
	resp := &Response{}

	buildDir, err := findModuleRoot()
	if err != nil {
		return nil, err
	}

	safeName := strings.ReplaceAll(t.Name(), "/", "_")
	serverBin := filepath.Join(os.TempDir(), "ai-critic-server-stream-"+safeName)
	agentBin := filepath.Join(os.TempDir(), "remote-agent-stream-"+safeName)

	for _, spec := range []struct {
		out  string
		pkg  string
	}{
		{serverBin, "."},
		{agentBin, "./cmd/remote-agent"},
	} {
		cmd := exec.Command("go", "build", "-o", spec.out, spec.pkg)
		cmd.Dir = buildDir
		out, err := cmd.CombinedOutput()
		if err != nil {
			return nil, fmt.Errorf("build %s: %v\n%s", spec.pkg, err, string(out))
		}
		t.Cleanup(func() { os.Remove(spec.out) })
	}

	configHome, err := lib.CreateTestConfigHome()
	if err != nil {
		return nil, err
	}
	t.Cleanup(func() { os.RemoveAll(configHome) })
	os.Setenv(lib.EnvAI_CRITIC_HOME, configHome)
	t.Cleanup(func() { os.Unsetenv(lib.EnvAI_CRITIC_HOME) })

	credFile, err := lib.WriteTestCredentials(configHome)
	if err != nil {
		return nil, err
	}

	if err := os.MkdirAll(configHome, 0755); err != nil {
		return nil, err
	}
	wsproxy.SetTestConfigDir(configHome)
	t.Cleanup(func() { wsproxy.SetTestConfigDir("") })
	wsproxy.SetTestStubNetworkChecks(true)
	t.Cleanup(func() { wsproxy.SetTestStubNetworkChecks(false) })
	if req.UpstreamFetchDelayMs > 0 {
		wsproxy.SetTestUpstreamFetchDelay(time.Duration(req.UpstreamFetchDelayMs) * time.Millisecond)
		t.Cleanup(func() { wsproxy.SetTestUpstreamFetchDelay(0) })
	}

	cfg := &wsproxy.Config{
		UpstreamProxy: "http://proxy.internal:3128",
		ListenPort:    0,
		WSPath:        "/ws",
		UUID:          "00000000-0000-4000-8000-000000000001",
		Subdomain:     "ws",
		InstanceID:    "25b2a55939e4",
		AutoStart:     true,
		PublicURL:     "https://ws-25b2a55939e4.xhd2015.xyz",
	}
	if err := wsproxy.SaveTestConfig(cfg); err != nil {
		return nil, err
	}

	serverPort, err := pickFreePort(24780)
	if err != nil {
		return nil, err
	}
	resp.ServerPort = serverPort

	serverCmd := exec.Command(serverBin, "--port", strconv.Itoa(serverPort), "--credentials-file", credFile)
	serverCmd.Dir = configHome
	serverCmd.Env = lib.AppendTestServerEnv(os.Environ(), configHome)
	if err := serverCmd.Start(); err != nil {
		return nil, fmt.Errorf("start server: %w", err)
	}
	t.Cleanup(func() {
		if serverCmd.Process != nil {
			serverCmd.Process.Signal(syscall.SIGTERM)
			time.Sleep(200 * time.Millisecond)
			serverCmd.Process.Kill()
		}
	})

	pingURL := fmt.Sprintf("http://127.0.0.1:%d/ping", serverPort)
	if err := waitHTTPReady(pingURL, 30*time.Second); err != nil {
		return nil, err
	}

	serverURL := fmt.Sprintf("http://127.0.0.1:%d", serverPort)
	agentCmd := exec.Command(agentBin,
		"--server", serverURL,
		"--token", lib.TestPassword,
		"ws-proxy", "doctor",
	)
	agentCmd.Env = os.Environ()

	stdout, err := agentCmd.StdoutPipe()
	if err != nil {
		return nil, err
	}
	var stderr bytes.Buffer
	agentCmd.Stderr = &stderr

	if err := agentCmd.Start(); err != nil {
		return nil, fmt.Errorf("start remote-agent: %w", err)
	}

	var lines []string
	var records []LineRecord
	scanner := bufio.NewScanner(stdout)
	scanner.Buffer(make([]byte, 64*1024), 4*1024*1024)
	for scanner.Scan() {
		line := scanner.Text()
		lines = append(lines, line)
		if req.RecordLineTimes {
			records = append(records, LineRecord{Line: line, At: time.Now()})
		}
	}
	_ = stdout.Close()
	waitErr := agentCmd.Wait()
	if waitErr != nil {
		if exitErr, ok := waitErr.(*exec.ExitError); ok {
			resp.ExitCode = exitErr.ExitCode()
		} else {
			return nil, waitErr
		}
	}

	resp.StdoutLines = lines
	resp.Stdout = strings.Join(lines, "\n")
	if stderr.Len() > 0 {
		resp.Stdout += "\n" + stderr.String()
	}
	resp.LineRecords = records

	resp.HasDoctorHdr = containsLine(resp.StdoutLines, "WS Proxy Doctor")
	resp.HasCheckLines = hasCheckPrefix(lines)
	resp.HasResultLine = containsLine(resp.StdoutLines, "Result: healthy") ||
		containsLine(resp.StdoutLines, "Result: unhealthy")

	return resp, nil
}

func findModuleRoot() (string, error) {
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
			return "", fmt.Errorf("go.mod not found")
		}
		dir = parent
	}
}

func pickFreePort(base int) (int, error) {
	for port := base; port < base+200; port++ {
		ln, err := net.Listen("tcp", fmt.Sprintf("127.0.0.1:%d", port))
		if err == nil {
			ln.Close()
			return port, nil
		}
	}
	return 0, fmt.Errorf("no free port near %d", base)
}

func waitHTTPReady(url string, timeout time.Duration) error {
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		resp, err := http.Get(url)
		if err == nil {
			resp.Body.Close()
			if resp.StatusCode == http.StatusOK {
				return nil
			}
		}
		time.Sleep(300 * time.Millisecond)
	}
	return fmt.Errorf("timeout waiting for %s", url)
}

func containsLine(lines []string, substr string) bool {
	for _, l := range lines {
		if strings.Contains(l, substr) {
			return true
		}
	}
	return false
}

func hasCheckPrefix(lines []string) bool {
	for _, l := range lines {
		if strings.Contains(l, "[ok]") || strings.Contains(l, "[fail]") || strings.Contains(l, "[skip]") || strings.Contains(l, "[warn]") {
			return true
		}
	}
	return false
}

func lineTime(records []LineRecord, substr string) (time.Time, bool) {
	for _, r := range records {
		if strings.Contains(r.Line, substr) {
			return r.At, true
		}
	}
	return time.Time{}, false
}
```

