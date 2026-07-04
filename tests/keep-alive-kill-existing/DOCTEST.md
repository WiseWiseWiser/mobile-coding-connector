# Keep-Alive `--kill-existing` Doctests

End-to-end tests for the `ai-critic keep-alive --kill-existing` flag: terminate
existing listeners on the keep-alive management port (`23312`) and the managed
server port before daemon startup, or preserve legacy port-in-use errors when
the flag is absent.

# DSN (Domain Specific Notion)

**Participants**

- **ai-critic binary** — built from repo root; invoked as `keep-alive` with
  optional `--kill-existing`, `--port`, `--forever`, and `--log`.
- **Port occupier subprocess** — minimal HTTP listener (`testdata/port-occupier`)
  bound to server and/or daemon ports before keep-alive starts; server occupier
  answers `GET /ping` with `pong` so `IsPortInUse` detects conflict.
- **Keep-alive daemon** — binds management port `23312`, spawns managed server
  on `--port`, exposes `GET /api/keep-alive/status`.
- **Test config home** — isolated `AI_CRITIC_HOME` with credentials; extension
  skipped via `AI_CRITIC_TEST_SKIP_EXTENSION=1` for fast, deterministic startup.
- **Session lock** — file lock on keep-alive management port so parallel doctest
  runs do not collide on `23312`.

**Behaviors**

- With `--kill-existing`, occupiers on daemon port and/or server port are
  SIGTERM'd (SIGKILL if needed) before the daemon binds and starts the server.
- With free ports and `--kill-existing`, daemon status reports running and
  management API responds on `23312`.
- Without `--kill-existing`, an occupied server port causes immediate startup
  error (no daemon management listener).

## Version

0.0.2

## Decision Tree

```
[keep-alive --kill-existing]
 |
 +-- no-conflict/                    (GROUP)  ports free, flag set
 |    +-- starts-cleanly/            (LEAF)   daemon status running
 |
 +-- server-port-occupied/           (GROUP)  server port listener only
 |    +-- kills-and-starts/          (LEAF)   occupier killed, daemon up
 |
 +-- daemon-port-occupied/           (GROUP)  management port listener only
 |    +-- kills-and-starts/          (LEAF)   occupier killed, API responds
 |
 +-- both-occupied/                  (GROUP)  both ports taken
 |    +-- kills-both/                (LEAF)   both occupiers killed, clean start
 |
 +-- no-flag/                        (GROUP)  legacy behavior without flag
      +-- port-occupied-errors/      (LEAF)   server port taken → startup error
```

## Test Index

| # | Leaf | Description |
|---|------|-------------|
| 1 | `no-conflict/starts-cleanly` | Free ports + `--kill-existing` → daemon running |
| 2 | `server-port-occupied/kills-and-starts` | Server port occupier killed; daemon starts |
| 3 | `daemon-port-occupied/kills-and-starts` | Daemon port occupier killed; status API OK |
| 4 | `both-occupied/kills-both` | Both occupiers killed; clean start |
| 5 | `no-flag/port-occupied-errors` | No flag + occupied server port → error, no start |

## Parameter Coverage

| Leaf | KillExisting | OccupyServer | OccupyDaemon | ExpectStart | ExpectError |
|------|--------------|--------------|--------------|-------------|-------------|
| starts-cleanly | true | false | false | true | false |
| kills-and-starts (server) | true | true | false | true | false |
| kills-and-starts (daemon) | true | false | true | true | false |
| kills-both | true | true | true | true | false |
| port-occupied-errors | false | true | false | false | true |

## How to Run

```sh
doctest vet ./tests/keep-alive-kill-existing
doctest test ./tests/keep-alive-kill-existing/...
```

```go
import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"
	"testing"
	"time"

	"github.com/xhd2015/ai-critic/run/daemon"
	"github.com/xhd2015/ai-critic/script/lib"
	"github.com/xhd2015/ai-critic/server/config"
)

type Request struct {
	ServerPort       int
	KillExisting     bool
	OccupyServerPort bool
	OccupyDaemonPort bool
	ExpectStart      bool
	ExpectError      bool
	StartupWaitSecs  int
}

type keepAliveStatus struct {
	Running       bool `json:"running"`
	ServerPort    int  `json:"server_port"`
	KeepAlivePort int  `json:"keep_alive_port"`
	KeepAlivePID  int  `json:"keep_alive_pid"`
	ServerPID     int  `json:"server_pid"`
}

type Response struct {
	ServerPort           int
	DaemonPort           int
	KillExisting         bool
	RunErr               string
	DaemonStarted        bool
	DaemonStatus         *keepAliveStatus
	OccupierServerPID    int
	OccupierDaemonPID    int
	OccupierServerKilled bool
	OccupierDaemonKilled bool
	CombinedOutput       string
}

func Run(t *testing.T, req *Request) (*Response, error) {
	resp := &Response{
		DaemonPort:   config.KeepAlivePort,
		KillExisting: req.KillExisting,
	}

	if req.ServerPort <= 0 {
		req.ServerPort = config.DefaultServerPort
	}
	hash := portHash(t.Name())
	req.ServerPort += hash % 200
	resp.ServerPort = req.ServerPort

	if req.StartupWaitSecs <= 0 {
		req.StartupWaitSecs = 15
	}

	moduleRoot, err := findModuleRoot()
	if err != nil {
		return nil, err
	}

	safeName := strings.ReplaceAll(t.Name(), "/", "_")
	binPath := filepath.Join(os.TempDir(), "ai-critic-kill-existing-"+safeName)
	build := exec.Command("go", "build", "-o", binPath, ".")
	build.Dir = moduleRoot
	if out, err := build.CombinedOutput(); err != nil {
		return nil, fmt.Errorf("build ai-critic: %v\n%s", err, string(out))
	}
	t.Cleanup(func() { os.Remove(binPath) })

	occupierBin, err := buildPortOccupier(moduleRoot)
	if err != nil {
		return nil, err
	}
	t.Cleanup(func() { os.Remove(occupierBin) })

	var occupierServerCmd, occupierDaemonCmd *exec.Cmd
	if req.OccupyServerPort {
		occupierServerCmd, resp.OccupierServerPID, err = startPortOccupier(occupierBin, req.ServerPort, true)
		if err != nil {
			return nil, fmt.Errorf("start server port occupier: %w", err)
		}
		t.Cleanup(stopOccupier(occupierServerCmd))
		if err := waitPortListening(req.ServerPort, 5*time.Second); err != nil {
			return nil, err
		}
	}
	if req.OccupyDaemonPort {
		occupierDaemonCmd, resp.OccupierDaemonPID, err = startPortOccupier(occupierBin, config.KeepAlivePort, false)
		if err != nil {
			return nil, fmt.Errorf("start daemon port occupier: %w", err)
		}
		t.Cleanup(stopOccupier(occupierDaemonCmd))
		if err := waitPortListening(config.KeepAlivePort, 5*time.Second); err != nil {
			return nil, err
		}
	}

	configHome, err := lib.CreateTestConfigHome()
	if err != nil {
		return nil, err
	}
	t.Cleanup(func() { os.RemoveAll(configHome) })
	if _, err := lib.WriteTestCredentials(configHome); err != nil {
		return nil, err
	}

	daemonLogPath := filepath.Join(configHome, "keep-alive-kill-existing.log")
	portStr := strconv.Itoa(req.ServerPort)
	daemonArgs := []string{
		"keep-alive",
		"--port", portStr,
		"--log", daemonLogPath,
	}
	if req.ExpectStart || req.KillExisting {
		daemonArgs = append(daemonArgs, "--forever")
	}
	if req.KillExisting {
		daemonArgs = append(daemonArgs, "--kill-existing")
	}

	cmd := exec.Command(binPath, daemonArgs...)
	cmd.Dir = configHome
	cmd.Env = buildDaemonEnv(configHome)
	var outBuf bytes.Buffer
	cmd.Stdout = &outBuf
	cmd.Stderr = &outBuf

	startErr := cmd.Start()
	if startErr != nil {
		resp.RunErr = startErr.Error()
		if req.ExpectError {
			return resp, nil
		}
		return resp, startErr
	}

	daemonPID := cmd.Process.Pid
	stopDaemon := func() {
		if cmd.Process != nil {
			pgid, pgErr := syscall.Getpgid(cmd.Process.Pid)
			if pgErr == nil {
				_ = syscall.Kill(-pgid, syscall.SIGTERM)
				time.Sleep(300 * time.Millisecond)
				_ = syscall.Kill(-pgid, syscall.SIGKILL)
			} else {
				_ = cmd.Process.Kill()
			}
		}
		_ = cmd.Wait()
	}
	t.Cleanup(stopDaemon)

	deadline := time.Now().Add(time.Duration(req.StartupWaitSecs) * time.Second)
	var status *keepAliveStatus
	for time.Now().Before(deadline) {
		if req.ExpectError {
			if err := cmd.Wait(); err != nil {
				resp.RunErr = err.Error()
				if exitErr, ok := err.(*exec.ExitError); ok {
					resp.RunErr = fmt.Sprintf("exit %d: %s", exitErr.ExitCode(), strings.TrimSpace(outBuf.String()))
				}
				logBytes, _ := os.ReadFile(daemonLogPath)
				resp.CombinedOutput = outBuf.String() + string(logBytes)
				resp.OccupierServerKilled = occupierDead(resp.OccupierServerPID)
				resp.OccupierDaemonKilled = occupierDead(resp.OccupierDaemonPID)
				return resp, nil
			}
		} else {
			st, stErr := fetchKeepAliveStatus()
			if stErr == nil && st != nil {
				status = st
				if st.KeepAlivePID > 0 && (st.Running || st.ServerPID > 0) {
					break
				}
			}
		}
		time.Sleep(250 * time.Millisecond)
	}

	if !req.ExpectError {
		if err := cmd.Process.Signal(syscall.Signal(0)); err != nil {
			waitErr := cmd.Wait()
			if waitErr != nil {
				resp.RunErr = waitErr.Error()
			}
		}
	}

	logBytes, _ := os.ReadFile(daemonLogPath)
	resp.CombinedOutput = outBuf.String() + string(logBytes)
	resp.DaemonStatus = status
	resp.DaemonStarted = status != nil && status.KeepAlivePID > 0
	if resp.DaemonStarted && daemonPID > 0 && status.KeepAlivePID == daemonPID {
		resp.DaemonStarted = true
	}
	resp.OccupierServerKilled = occupierDead(resp.OccupierServerPID)
	resp.OccupierDaemonKilled = occupierDead(resp.OccupierDaemonPID)

	if req.ExpectError && resp.RunErr == "" {
		resp.RunErr = "expected startup error but keep-alive did not fail"
	}

	return resp, nil
}

func buildDaemonEnv(configHome string) []string {
	env := lib.AppendTestServerEnv(os.Environ(), configHome)
	env = append(env, "AI_CRITIC_TEST_SKIP_EXTENSION=1")
	return env
}

func fetchKeepAliveStatus() (*keepAliveStatus, error) {
	url := fmt.Sprintf("http://127.0.0.1:%d/api/keep-alive/status", config.KeepAlivePort)
	resp, err := http.Get(url)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("status %d: %s", resp.StatusCode, strings.TrimSpace(string(body)))
	}
	var out keepAliveStatus
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		return nil, err
	}
	return &out, nil
}

func occupierDead(pid int) bool {
	return daemon.TestExported_OccupierDead(pid)
}

func processAlive(pid int) bool {
	if pid <= 0 {
		return false
	}
	return syscall.Kill(pid, 0) == nil
}

func portHash(name string) int {
	hash := 0
	for _, c := range name {
		hash = hash*31 + int(c)
	}
	if hash < 0 {
		hash = -hash
	}
	return hash
}

func findModuleRoot() (string, error) {
	if root := os.Getenv("DOCTEST_ROOT"); root != "" {
		for dir := root; ; dir = filepath.Dir(dir) {
			if _, err := os.Stat(filepath.Join(dir, "go.mod")); err == nil {
				return dir, nil
			}
			if filepath.Dir(dir) == dir {
				break
			}
		}
	}
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

func buildPortOccupier(moduleRoot string) (string, error) {
	doctestRoot := filepath.Join(moduleRoot, "tests", "keep-alive-kill-existing")
	src := filepath.Join(doctestRoot, "testdata", "port-occupier")
	out := filepath.Join(os.TempDir(), "port-occupier-"+strconv.Itoa(os.Getpid()))
	cmd := exec.Command("go", "build", "-o", out, ".")
	cmd.Dir = src
	if combined, err := cmd.CombinedOutput(); err != nil {
		return "", fmt.Errorf("build port-occupier: %v\n%s", err, string(combined))
	}
	return out, nil
}

func startPortOccupier(bin string, port int, withPing bool) (*exec.Cmd, int, error) {
	args := []string{"--port", strconv.Itoa(port)}
	if withPing {
		args = append(args, "--ping")
	}
	cmd := exec.Command(bin, args...)
	cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}
	if err := cmd.Start(); err != nil {
		return nil, 0, err
	}
	return cmd, cmd.Process.Pid, nil
}

func stopOccupier(cmd *exec.Cmd) func() {
	return func() {
		if cmd == nil || cmd.Process == nil {
			return
		}
		pgid, err := syscall.Getpgid(cmd.Process.Pid)
		if err == nil {
			_ = syscall.Kill(-pgid, syscall.SIGKILL)
		} else {
			_ = cmd.Process.Kill()
		}
		_ = cmd.Wait()
	}
}

func waitPortListening(port int, timeout time.Duration) error {
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		if pid := daemon.FindPortPID(port); pid != "" {
			return nil
		}
		time.Sleep(100 * time.Millisecond)
	}
	return fmt.Errorf("timeout waiting for listener on port %d", port)
}
```