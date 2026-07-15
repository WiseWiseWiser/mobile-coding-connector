# Remote-Agent Paste-Bin Doctests

End-to-end tests for `remote-agent paste-bin`: read/write the File Transfer Quick
Transfer scratch pad via `GET/PUT /api/file-transfer/scratch`.

# DSN (Domain Specific Notion)

The harness exercises the remote-agent CLI against a real `ai-critic-server`
subprocess with scratch storage under isolated `AI_CRITIC_HOME/file-transfer/scratch.json`.

**Participants**

- **remote-agent subprocess** — built from `./cmd/remote-agent`; `paste-bin` reads
  or writes scratch via the HTTP client.
- **HTTP client** — `GET/PUT /api/file-transfer/scratch` for scratch content and metadata.
- **ai-critic-server subprocess** — ephemeral port; `AI_CRITIC_HOME=configHome`; serves
  scratch API against `{configHome}/file-transfer/scratch.json`.
- **configHome** — temp isolated server config; leaf setup seeds or deletes scratch.json.
- **agentHome** — temp `HOME` with `~/.ai-critic/remote-agent-config.json` only.
- **stdin pipe** — write leaves attach piped stdin (`os.Stdin` not a character device);
  read leaves use TTY (no stdin attachment) unless testing `--read` override.
- **session cache** — doctest-injected `DOCTEST_SESSION_ID` keys
  `$TMPDIR/remote-agent-paste-bin-doctest-<id>/` for shared binaries (file lock).

**Behaviors**

- TTY stdin (default): **read** — GET scratch, print `content` to stdout; empty scratch
  → silent stdout, exit 0.
- Piped stdin (default): **write** — read raw stdin bytes, PUT scratch (overwrite);
  empty pipe clears scratch (`content: ""`).
- `--read` with piped stdin: force read; stdin bytes ignored for write.
- Write success stderr: green `saved N bytes`, gray preview block (max 3 lines / 200 bytes);
  truncation hint when preview shorter than payload.
- Write stdout echo: full content when `1 ≤ N ≤ 4096`; silent when `N=0` or `N>4096`.
- Invalid UTF-8 stdin: stored as `paste-bin:b64:<base64>` envelope; read decodes to raw bytes.
- `--json`: JSON on stdout only (read or write); no ANSI, no preview/echo.
- `--meta`: gray `updated at` (read) or `saved at` (write) timestamp on stderr.
- `-q` / `--quiet`: suppress stdout echo on small writes.
- Bad token or extra positional args: non-zero exit with actionable errors.

## Version

0.0.2

## Decision Tree

```
[remote-agent paste-bin]
 |
 +-- read/                              (GROUP) TTY read mode
 |    +-- empty/                        (LEAF)  no scratch.json → silent stdout
 |    +-- seeded-utf8/                  (LEAF)  multiline UTF-8 bytes on stdout
 |    +-- json-flag/                    (LEAF)  --json → JSON stdout
 |    +-- meta-flag/                    (LEAF)  --meta → timestamp on stderr
 |
 +-- write/                             (GROUP) piped stdin write mode
 |    +-- small-echo/                   (LEAF)  ≤4096 bytes → stderr saved + stdout echo
 |    +-- empty-pipe/                   (LEAF)  empty stdin clears scratch
 |    +-- large-no-echo/                (LEAF)  >4096 bytes → preview only, no stdout
 |    +-- binary-envelope/              (LEAF)  NUL bytes → b64 envelope round-trip
 |    +-- json-flag/                    (LEAF)  --json → PUT JSON on stdout only
 |    +-- quiet-flag/                   (LEAF)  -q suppresses stdout echo
 |
 +-- mode-override/                     (GROUP) flag precedence over stdin detection
 |    +-- read-force-piped/             (LEAF)  piped stdin + --read → read, API unchanged
 |
 +-- rejected/                          (GROUP) CLI validation and auth errors
      +-- extra-args/                  (LEAF)  positional args rejected
      +-- auth-failure/                 (LEAF)  bad token → non-zero exit
```

## Test Index

| # | Leaf | Description |
|---|------|-------------|
| 1 | `read/empty` | Missing scratch.json; exit 0; silent stdout |
| 2 | `read/seeded-utf8` | Seeded multiline UTF-8; stdout matches bytes |
| 3 | `read/json-flag` | `--json` prints scratch JSON on stdout |
| 4 | `read/meta-flag` | `--meta` prints gray `updated at` on stderr |
| 5 | `write/small-echo` | Pipe `hi`; stderr `saved 2 bytes`; stdout echoes `hi` |
| 6 | `write/empty-pipe` | Empty pipe clears scratch; `saved 0 bytes` |
| 7 | `write/large-no-echo` | 5000-byte pipe; stderr preview + truncation; silent stdout |
| 8 | `write/binary-envelope` | NUL stdin; API b64 envelope; decoded bytes match |
| 9 | `write/json-flag` | `--json` on write; JSON only on stdout |
| 10 | `write/quiet-flag` | `-q` suppresses stdout echo for small write |
| 11 | `mode-override/read-force-piped` | Piped junk + `--read`; seeded stdout; API unchanged |
| 12 | `rejected/extra-args` | `paste-bin foo` → non-zero; usage/args hint |
| 13 | `rejected/auth-failure` | Bad `--token` → non-zero exit |

## Parameter Coverage

| Factor | Leaves |
|--------|--------|
| Operation mode (read vs write) | read/*, write/* |
| Stdin source (TTY vs pipe vs empty pipe) | read/*, write/*, mode-override/* |
| `--read` override | mode-override/read-force-piped |
| Output flags (`--json`, `--meta`, `-q`) | read/json-flag, read/meta-flag, write/json-flag, write/quiet-flag |
| Payload size (0, small ≤4096, large >4096) | read/empty, write/small-echo, write/empty-pipe, write/large-no-echo |
| Binary / UTF-8 encoding | read/seeded-utf8, write/binary-envelope |
| Scratch seed state (absent, seeded, stale) | read/empty, read/*, write/empty-pipe, mode-override/* |
| Auth / CLI surface errors | rejected/* |

## How to Run

```sh
go run ./script/build
doctest vet ./tests/remote-agent-paste-bin
doctest test ./tests/remote-agent-paste-bin/...
```

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

	"github.com/xhd2015/ai-critic/script/lib"
)

// ScratchSeed configures scratch.json before the CLI runs.
type ScratchSeed struct {
	Content   string
	UpdatedAt string
}

type Request struct {
	Args   []string
	Server string
	Token  string

	ScratchReset bool
	ScratchSeed  *ScratchSeed

	// PipedStdin nil = TTY (stdin not attached). Non-nil = pipe stdin using StdinBytes
	// (may be empty slice for empty-pipe clears).
	PipedStdin *bool
	StdinBytes []byte
}

type ScratchEntry struct {
	Content   string `json:"content"`
	UpdatedAt string `json:"updated_at"`
}

type Response struct {
	ExitCode   int
	Stdout     string
	Stderr     string
	Combined   string
	ServerPort int
	ServerURL  string
	ConfigHome string
	AgentHome  string
	Token      string

	ScratchAfter ScratchEntry
}

func Run(t *testing.T, req *Request) (*Response, error) {
	resp := &Response{}

	if len(req.Args) == 0 {
		req.Args = []string{"paste-bin"}
	}
	if req.Token == "" {
		req.Token = lib.TestPassword
	}
	resp.Token = req.Token

	moduleRoot := findModuleRoot()
	cacheDir := sessionCacheDir()
	serverBin, agentBin := buildSessionBinariesOnce(t, moduleRoot, cacheDir)

	configHome, err := lib.CreateTestConfigHome()
	if err != nil {
		return nil, err
	}
	t.Cleanup(func() { os.RemoveAll(configHome) })
	resp.ConfigHome = configHome

	if err := prepareScratch(req, configHome); err != nil {
		return nil, fmt.Errorf("prepare scratch: %w", err)
	}

	agentHome, err := os.MkdirTemp("", "remote-agent-paste-bin-home-*")
	if err != nil {
		return nil, err
	}
	t.Cleanup(func() { os.RemoveAll(agentHome) })
	resp.AgentHome = agentHome

	credFile, err := lib.WriteTestCredentials(configHome)
	if err != nil {
		return nil, err
	}

	portBase := portBaseFromTestName(t.Name())
	serverPort := pickFreePort(portBase)
	resp.ServerPort = serverPort

	serverURL := req.Server
	if serverURL == "" {
		serverURL = fmt.Sprintf("http://localhost:%d", serverPort)
	}
	normalizedServer := strings.TrimRight(strings.TrimSpace(serverURL), "/")
	resp.ServerURL = normalizedServer

	configPath := filepath.Join(agentHome, ".ai-critic", "remote-agent-config.json")
	if err := writeRemoteAgentConfig(configPath, normalizedServer, req.Token); err != nil {
		return nil, err
	}

	serverCmd := exec.Command(serverBin, "--port", strconv.Itoa(serverPort), "--credentials-file", credFile)
	serverCmd.Dir = configHome
	serverCmd.Env = lib.AppendTestServerEnv(os.Environ(), configHome)
	if err := serverCmd.Start(); err != nil {
		return nil, fmt.Errorf("start server: %w", err)
	}
	t.Cleanup(func() {
		if serverCmd.Process != nil {
			serverCmd.Process.Signal(syscall.SIGTERM)
			time.Sleep(150 * time.Millisecond)
			serverCmd.Process.Kill()
		}
	})

	pingURL := fmt.Sprintf("http://127.0.0.1:%d/ping", serverPort)
	if err := waitHTTPReady(pingURL, 30*time.Second); err != nil {
		return nil, err
	}

	argv := []string{"--server", serverURL, "--token", req.Token}
	argv = append(argv, req.Args...)
	t.Logf("remote-agent argv: %v", argv)

	agentCmd := exec.Command(agentBin, argv...)
	agentEnv := stripEnvPrefix(os.Environ(), "HOME=")
	agentEnv = append(agentEnv, "HOME="+agentHome)
	agentCmd.Env = agentEnv

	if req.PipedStdin != nil {
		agentCmd.Stdin = bytes.NewReader(req.StdinBytes)
	}

	var stdout, stderr bytes.Buffer
	agentCmd.Stdout = &stdout
	agentCmd.Stderr = &stderr

	runErr := agentCmd.Run()
	if runErr != nil {
		if exitErr, ok := runErr.(*exec.ExitError); ok {
			resp.ExitCode = exitErr.ExitCode()
		} else {
			return nil, runErr
		}
	}
	resp.Stdout = stdout.String()
	resp.Stderr = stderr.String()
	resp.Combined = strings.TrimSpace(resp.Stdout + "\n" + resp.Stderr)

	// Post-CLI verification always uses server credentials so auth-failure leaves
	// can still assert scratch state was not mutated.
	entry, err := fetchScratchEntry(normalizedServer, lib.TestPassword)
	if err != nil {
		return nil, fmt.Errorf("fetch scratch after CLI: %w", err)
	}
	resp.ScratchAfter = entry

	return resp, nil
}

type remoteAgentConfigFile struct {
	Default string            `json:"default,omitempty"`
	Domains []domainConfigRow `json:"domains"`
}

type domainConfigRow struct {
	Server string `json:"server"`
	Token  string `json:"token,omitempty"`
}

func writeRemoteAgentConfig(path, server, token string) error {
	cfg := remoteAgentConfigFile{
		Default: server,
		Domains: []domainConfigRow{{Server: server, Token: token}},
	}
	data, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		return err
	}
	return os.WriteFile(path, data, 0600)
}

func findModuleRoot() string {
	dir, err := os.Getwd()
	if err != nil {
		panic(err)
	}
	for {
		if _, err := os.Stat(filepath.Join(dir, "go.mod")); err == nil {
			return dir
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			panic("go.mod not found")
		}
		dir = parent
	}
}

func portBaseFromTestName(name string) int {
	hash := 0
	for _, c := range name {
		hash = hash*31 + int(c)
	}
	if hash < 0 {
		hash = -hash
	}
	return 27000 + (hash % 1000)
}

func pickFreePort(base int) int {
	for port := base; port < base+200; port++ {
		ln, err := net.Listen("tcp", fmt.Sprintf("127.0.0.1:%d", port))
		if err == nil {
			ln.Close()
			return port
		}
	}
	panic(fmt.Sprintf("no free port near %d", base))
}

func stripEnvPrefix(env []string, prefix string) []string {
	out := make([]string, 0, len(env))
	for _, e := range env {
		if strings.HasPrefix(e, prefix) {
			continue
		}
		out = append(out, e)
	}
	return out
}

func waitHTTPReady(url string, timeout time.Duration) error {
	deadline := time.Now().Add(timeout)
	client := &http.Client{Timeout: 2 * time.Second}
	for time.Now().Before(deadline) {
		resp, err := client.Get(url)
		if err == nil {
			resp.Body.Close()
			if resp.StatusCode == http.StatusOK {
				return nil
			}
		}
		time.Sleep(200 * time.Millisecond)
	}
	return fmt.Errorf("timeout waiting for %s", url)
}

func fetchScratchEntry(serverURL, token string) (ScratchEntry, error) {
	req, err := http.NewRequest(http.MethodGet, strings.TrimRight(serverURL, "/")+"/api/file-transfer/scratch", nil)
	if err != nil {
		return ScratchEntry{}, err
	}
	req.Header.Set("Authorization", "Bearer "+token)
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return ScratchEntry{}, err
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return ScratchEntry{}, err
	}
	if resp.StatusCode != http.StatusOK {
		return ScratchEntry{}, fmt.Errorf("GET scratch status %d: %s", resp.StatusCode, strings.TrimSpace(string(body)))
	}
	var entry ScratchEntry
	if err := json.Unmarshal(body, &entry); err != nil {
		return ScratchEntry{}, err
	}
	return entry, nil
}
```