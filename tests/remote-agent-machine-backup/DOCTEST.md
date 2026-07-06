# Remote-Agent Machine Backup & Restore Doctests

End-to-end tests for `remote-agent machine backup` and `remote-agent machine restore`:
server-home dot-file/dot-dir discovery, built-in and custom exclusions, SSE
streaming dry-run plans (per-entry sizes + summary), streamed `tar.xz` archives
with `manifest.json`, and restore with identical skip reporting.

# DSN (Domain Specific Notion)

The harness models a remote machine as an isolated `serverHome` directory. The
`ai-critic-server` subprocess runs with `HOME=serverHome` and working directory
`serverHome`, so server `~` and backup scope align. The CLI runs in a separate
`agentHome` with only `remote-agent-config.json`. Backup walks direct children of
server home (dot-files and dot-dirs), applies built-in exclusions plus optional
`--exclude`, archives symlinks without following, and streams `tar.xz` for real
backups. With `--dry-run`, backup and restore use SSE `/stream` endpoints:
incremental per-entry lines (with sizes on backup) followed by a human summary
block after the `done` frame. Restore reads the archive, skips byte-identical
entries (printing `skip (identical): <path>` in dry-run and apply), and applies
create/update actions.

**Participants**

- **remote-agent subprocess** — `./cmd/remote-agent`; subcommands `machine backup`
  and `machine restore` with `--server` / `--token`.
- **ai-critic-server subprocess** — ephemeral port; `POST /api/remote-agent/machine/backup`
  (`application/x-xz` stream) and `POST /api/remote-agent/machine/backup/stream` (SSE
  dry-run plan); `POST /api/remote-agent/machine/restore` (apply) and
  `POST /api/remote-agent/machine/restore/stream?dry_run=true` (SSE dry-run plan).
- **serverHome** — temp fake machine home seeded with dot fixtures and built-in
  exclusion trees (`.cache`, `.npm`, `.cargo/registry`, etc.).
- **agentHome** — temp `HOME` for `~/.ai-critic/remote-agent-config.json` only.
- **session cache** — doctest-injected `DOCTEST_SESSION_ID` keys
  `$TMPDIR/machine-backup-doctest-<id>/` for shared binaries and a default prereq
  archive (file lock + flock). Helpers use the variable directly, not `os.Getenv`.

**Behaviors**

- `machine backup --dry-run` streams DOT FILES / DOT DIRS / EXCLUDED sections with
  per-entry sizes and exclusion reasons, then prints `dry-run: machine backup plan`
  summary; no archive file is written.
- `machine backup --show-config` prints built-in exclusion config JSON locally (no
  server backup API call).
- `machine backup` (default) streams `tar.xz` containing `manifest.json`, included
  paths relative to server home, and phantom `.backup/` meta entries injected at
  pack time (`config.json`, `installed.json`, `ENV`, and optional `*.machine.bak`
  snapshots of pre-existing `~/.backup/*` files).
- Repeatable `--exclude` merges with built-in exclusions; repeatable `--include`
  re-includes built-in excluded paths. Effective rule: `(defaults − include) ∪ exclude`.
- `machine restore --dry-run` streams `skip (identical):` / `update:` / `create:` lines,
  then prints `dry-run: machine restore plan` summary with counts; no writes.
- `machine restore` applies create/update entries; identical paths are skipped with
  the same skip line printed to stdout.
- `machine restore --show-config` without archive prints built-in config JSON; with
  archive prints `.backup/config.json` from the archive (or built-in fallback).
- `machine restore --show-meta` requires an archive and prints `.backup/*` meta
  except `config.json` and `*.machine.bak`.
- Restore skips meta snapshots (`.backup/config.json`, `.backup/installed.json`,
  `.backup/ENV`) but restores `.backup/*.machine.bak` to `~/.backup/{original name}`.

## Version

0.0.2

## Decision Tree

```
[remote-agent machine backup | restore]
 |
 +-- backup/                              (GROUP)  snapshot server HOME
 |    |
 |    +-- dry-run/                        (LEAF)   plan only; rollups + exclusion reasons
 |    +-- stream/                         (LEAF)   tar.xz stream; manifest + members
 |    +-- custom-exclude/                 (LEAF)   --exclude drops extra dot-dir
 |    +-- show-config/                    (LEAF)   built-in exclusion config JSON
 |    +-- include/                        (LEAF)   --include re-includes built-in path
 |    +-- backup-meta/                    (LEAF)   archive .backup/ meta + machine.bak
 |
 +-- restore/                             (GROUP)  apply archive to server HOME
      |
      +-- dry-run-identical/              (LEAF)   unchanged home → skip lines only
      +-- dry-run-changed/                (LEAF)   mutated file → update in plan
      +-- apply/                          (LEAF)   writes changes; skips identical
      +-- show-config-builtin/            (LEAF)   built-in config JSON (no archive)
      +-- show-config-archive/            (LEAF)   effective config from archive
      +-- show-meta/                      (LEAF)   installed.json + ENV sections
      +-- meta-restore/                   (LEAF)   .machine.bak restores ~/.backup/*
```

## Test Index

| # | Leaf | Description |
|---|------|-------------|
| 1 | `backup/dry-run` | Streamed plan with sizes and exclusion reasons; summary rollups |
| 2 | `backup/stream` | Writes valid `tar.xz` with manifest and included members |
| 3 | `backup/custom-exclude` | `--exclude .docker` omits `.docker` from plan and archive |
| 4 | `backup/show-config` | `--show-config` prints built-in exclusion JSON; no backup API |
| 5 | `backup/include` | `--include .cache` re-includes `.cache` tree in dry-run plan |
| 6 | `backup/backup-meta` | Archive contains `.backup/` meta; seeded config → `.machine.bak` |
| 7 | `restore/dry-run-identical` | Home matches archive → skip stream + restore summary, no writes |
| 8 | `restore/dry-run-changed` | Modified `.bashrc` → `update:` stream line + restore summary counts |
| 9 | `restore/apply` | Apply restores changed file; identical paths still skipped |
| 10 | `restore/show-config-builtin` | `--show-config` without archive prints built-in JSON |
| 11 | `restore/show-config-archive` | Prereq backup → `--show-config` prints archive effective config |
| 12 | `restore/show-meta` | Prereq backup → `--show-meta` prints installed.json + ENV only |
| 13 | `restore/meta-restore` | Prereq backup with seeded meta → apply restores `.machine.bak` content |

## Parameter Coverage

| Factor | Leaves |
|--------|--------|
| Subcommand `backup` | backup/* |
| Subcommand `restore` | restore/* |
| `--dry-run` | backup/dry-run, backup/include, restore/dry-run-identical, restore/dry-run-changed |
| `--show-config` | backup/show-config, restore/show-config-builtin, restore/show-config-archive |
| `--show-meta` | restore/show-meta |
| Streamed archive (no dry-run) | backup/stream, backup/backup-meta, restore/* (prereq backup except show-config-builtin) |
| Built-in exclusions | backup/dry-run, backup/stream, backup/show-config, restore/show-config-builtin |
| Custom `--exclude` | backup/custom-exclude |
| Custom `--include` | backup/include, restore/meta-restore (`--include .backup` for machine.bak apply) |
| Archive `.backup/` meta | backup/backup-meta, restore/show-config-archive, restore/show-meta, restore/meta-restore |
| Seeded `~/.backup/config.json` | backup/backup-meta, restore/meta-restore |
| Identical vs changed restore target | restore/dry-run-identical, restore/dry-run-changed, restore/apply, restore/meta-restore |

## How to Run

```sh
go run ./script/build
doctest vet ./tests/remote-agent-machine-backup
doctest test -v ./tests/remote-agent-machine-backup/...
go test ./server/machinebackup/... -count=1
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

type Request struct {
	Args  []string
	Server string
	Token  string

	// OutputPath is the backup archive destination (backup leaves).
	OutputPath string

	// RestoreArchive is the tar.xz path passed to restore (set by Run when PrereqBackup).
	RestoreArchive string

	// ExcludePaths are appended as repeated --exclude flags before Args.
	ExcludePaths []string

	// IncludePaths are appended as repeated --include flags before Args.
	IncludePaths []string

	// PrereqBackup causes Run to execute `machine backup` before the main invocation.
	PrereqBackup bool

	// AfterBackupMutate selects post-backup server home changes for restore leaves.
	// Values: "" (none), "modify-bashrc", "wipe-backup-config".
	AfterBackupMutate string

	// SeedDocker adds .docker/config for custom-exclude coverage.
	SeedDocker bool

	// SeedBackupMeta seeds serverHome/.backup/config.json with distinguishable old JSON.
	SeedBackupMeta bool

	// ShowConfig and ShowMeta are set by leaves; Run appends flags when building argv.
	ShowConfig bool
	ShowMeta   bool
}

type Response struct {
	ExitCode   int
	Stdout     string
	Stderr     string
	Combined   string
	ServerPort int
	ServerHome string
	AgentHome  string

	BackupPath string
}

func Run(t *testing.T, req *Request) (*Response, error) {
	resp := &Response{}

	if req.Token == "" {
		req.Token = lib.TestPassword
	}

	moduleRoot := findModuleRoot()
	cacheDir := sessionCacheDir()
	serverBin, agentBin := buildSessionBinariesOnce(t, moduleRoot, cacheDir)

	serverHome, err := os.MkdirTemp("", "machine-backup-server-home-*")
	if err != nil {
		return nil, err
	}
	t.Cleanup(func() { os.RemoveAll(serverHome) })
	resp.ServerHome = serverHome

	if err := seedServerHome(t, serverHome, req.SeedDocker); err != nil {
		return nil, err
	}
	if req.SeedBackupMeta {
		seedBackupMeta(t, serverHome)
	}

	agentHome, err := os.MkdirTemp("", "machine-backup-agent-home-*")
	if err != nil {
		return nil, err
	}
	t.Cleanup(func() { os.RemoveAll(agentHome) })
	resp.AgentHome = agentHome

	credDir := filepath.Join(serverHome, ".ai-critic")
	if err := os.MkdirAll(credDir, 0755); err != nil {
		return nil, err
	}
	credFile := filepath.Join(credDir, "server-credentials")
	if err := os.WriteFile(credFile, []byte(req.Token+"\n"), 0600); err != nil {
		return nil, fmt.Errorf("write credentials: %w", err)
	}

	remoteConfigPath := filepath.Join(agentHome, ".ai-critic", "remote-agent-config.json")
	if err := os.MkdirAll(filepath.Dir(remoteConfigPath), 0755); err != nil {
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

	if err := writeRemoteAgentConfig(remoteConfigPath, normalizedServer, req.Token); err != nil {
		return nil, err
	}

	killPort(serverPort)

	serverCmd := exec.Command(serverBin, "--port", strconv.Itoa(serverPort), "--credentials-file", credFile)
	serverCmd.Dir = serverHome
	serverCmd.Env = stripEnvPrefix(os.Environ(), "HOME=")
	serverCmd.Env = stripEnvPrefix(serverCmd.Env, lib.EnvAI_CRITIC_HOME+"=")
	serverCmd.Env = append(serverCmd.Env, "HOME="+serverHome)
	serverCmd.Env = append(serverCmd.Env, "AI_CRITIC_NO_OPEN_BROWSER=1")
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
	if err := verifyServerHome(t, normalizedServer, req.Token, serverHome); err != nil {
		return nil, err
	}

	agentEnv := stripEnvPrefix(os.Environ(), "HOME=")
	agentEnv = append(agentEnv, "HOME="+agentHome)

	if req.PrereqBackup {
		var backupPath string
		if needsCustomPrereqArchive(req) {
			backupPath = filepath.Join(agentHome, "prereq-backup.tar.xz")
			backupArgs := []string{"--server", serverURL, "--token", req.Token, "machine", "backup", "--output", backupPath}
			for _, ex := range req.ExcludePaths {
				backupArgs = append(backupArgs, "--exclude", ex)
			}
			for _, inc := range req.IncludePaths {
				backupArgs = append(backupArgs, "--include", inc)
			}
			t.Logf("prereq backup argv: %v", backupArgs)
			if code, out, errOut, runErr := runAgent(agentBin, backupArgs, agentEnv); runErr != nil {
				return nil, runErr
			} else if code != 0 {
				return nil, fmt.Errorf("prereq backup exit %d:\n%s\n%s", code, out, errOut)
			}
		} else {
			var err error
			backupPath, err = ensureSessionDefaultArchive(t, moduleRoot, serverBin, agentBin, cacheDir, req.Token)
			if err != nil {
				return nil, err
			}
			t.Logf("prereq backup: reusing session archive %s", backupPath)
		}
		req.RestoreArchive = backupPath
		resp.BackupPath = backupPath

		switch req.AfterBackupMutate {
		case "":
		case "modify-bashrc":
			bashrcPath := filepath.Join(serverHome, ".bashrc")
			if err := os.WriteFile(bashrcPath, []byte("mutated after backup\n"), 0644); err != nil {
				return nil, err
			}
			if data, readErr := os.ReadFile(bashrcPath); readErr != nil {
				t.Logf("post-mutation serverHome/.bashrc read error: %v", readErr)
			} else {
				t.Logf("post-mutation serverHome/.bashrc: %q", string(data))
			}
		case "wipe-backup-config":
			writeServerFile(t, serverHome, ".backup/config.json", `{"wiped":true}`+"\n")
		default:
			return nil, fmt.Errorf("unknown AfterBackupMutate %q", req.AfterBackupMutate)
		}
	}

	argv := make([]string, 0, len(req.Args)+16)
	argv = append(argv, "--server", serverURL, "--token", req.Token)
	argv = append(argv, req.Args...)

	if req.RestoreArchive != "" {
		replaced := false
		for i, arg := range argv {
			if arg == "__RESTORE_ARCHIVE__" {
				argv[i] = req.RestoreArchive
				replaced = true
			}
		}
		if !replaced {
			argv = insertRestoreArchive(argv, req.RestoreArchive)
		}
	}

	var subcommandFlags []string
	for _, ex := range req.ExcludePaths {
		subcommandFlags = append(subcommandFlags, "--exclude", ex)
	}
	for _, inc := range req.IncludePaths {
		subcommandFlags = append(subcommandFlags, "--include", inc)
	}
	if req.ShowConfig && !argvContainsFlag(argv, "--show-config") {
		subcommandFlags = append(subcommandFlags, "--show-config")
	}
	if req.ShowMeta && !argvContainsFlag(argv, "--show-meta") {
		subcommandFlags = append(subcommandFlags, "--show-meta")
	}
	if len(subcommandFlags) > 0 {
		argv = insertSubcommandFlags(argv, subcommandFlags...)
	}

	if req.OutputPath != "" {
		out := req.OutputPath
		if !filepath.IsAbs(out) {
			out = filepath.Join(agentHome, out)
		}
		absOut, err := filepath.Abs(out)
		if err != nil {
			return nil, err
		}
		req.OutputPath = absOut
		for i, arg := range argv {
			if arg == "__OUTPUT_PATH__" {
				argv[i] = absOut
			}
		}
	}

	t.Logf("remote-agent argv: %v", argv)

	exitCode, stdout, stderr, runErr := runAgent(agentBin, argv, agentEnv)
	if runErr != nil {
		return nil, runErr
	}

	resp.ExitCode = exitCode
	resp.Stdout = stdout
	resp.Stderr = stderr
	resp.Combined = strings.TrimSpace(resp.Stdout + "\n" + resp.Stderr)

	if resp.BackupPath == "" && req.OutputPath != "" {
		resp.BackupPath = req.OutputPath
	}

	return resp, nil
}

func runAgent(bin string, argv, env []string) (int, string, string, error) {
	cmd := exec.Command(bin, argv...)
	cmd.Env = env
	var outBuf, errBuf bytes.Buffer
	cmd.Stdout = &outBuf
	cmd.Stderr = &errBuf
	runErr := cmd.Run()
	exitCode := 0
	if runErr != nil {
		if exitErr, ok := runErr.(*exec.ExitError); ok {
			exitCode = exitErr.ExitCode()
		} else {
			return 0, "", "", runErr
		}
	}
	return exitCode, outBuf.String(), errBuf.String(), nil
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
	return 26000 + (hash % 1000)
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

func killPort(port int) {
	out, err := exec.Command("lsof", "-ti", fmt.Sprintf(":%d", port)).Output()
	if err != nil {
		return
	}
	for _, pidStr := range strings.Fields(strings.TrimSpace(string(out))) {
		_ = exec.Command("kill", "-9", pidStr).Run()
	}
}

func normalizeAbsPath(path string) (string, error) {
	abs, err := filepath.Abs(path)
	if err != nil {
		return "", err
	}
	eval, err := filepath.EvalSymlinks(abs)
	if err != nil {
		return abs, nil
	}
	return eval, nil
}

func verifyServerHome(t *testing.T, serverURL, token, wantHome string) error {
	want, err := normalizeAbsPath(wantHome)
	if err != nil {
		return fmt.Errorf("resolve harness serverHome: %w", err)
	}
	backupURL := strings.TrimRight(strings.TrimSpace(serverURL), "/") + "/api/remote-agent/machine/backup"
	body := `{"dry_run":true,"exclude":[],"include":[]}`
	req, err := http.NewRequest(http.MethodPost, backupURL, strings.NewReader(body))
	if err != nil {
		return fmt.Errorf("build verify-home request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+token)
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return fmt.Errorf("verify server HOME: %w", err)
	}
	defer resp.Body.Close()
	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("read verify-home response: %w", err)
	}
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("verify server HOME status %d: %s", resp.StatusCode, strings.TrimSpace(string(data)))
	}
	var plan struct {
		Home string `json:"home"`
	}
	if err := json.Unmarshal(data, &plan); err != nil {
		return fmt.Errorf("decode backup plan for HOME verify: %w", err)
	}
	got, err := normalizeAbsPath(plan.Home)
	if err != nil {
		return fmt.Errorf("resolve server-reported HOME %q: %w", plan.Home, err)
	}
	if got != want {
		return fmt.Errorf(
			"server HOME mismatch on %s: server reports %q (normalized %q) but harness serverHome is %q (normalized %q); stale process may still be bound to the port",
			backupURL, plan.Home, got, wantHome, want,
		)
	}
	t.Logf("verified server HOME=%s", got)
	return nil
}

func waitHTTPReady(url string, timeout time.Duration) error {
	deadline := time.Now().Add(timeout)
	client := &http.Client{Timeout: 2 * time.Second}
	for time.Now().Before(deadline) {
		resp, err := client.Get(url)
		if err == nil {
			_, _ = io.Copy(io.Discard, resp.Body)
			resp.Body.Close()
			if resp.StatusCode == http.StatusOK {
				return nil
			}
		}
		time.Sleep(200 * time.Millisecond)
	}
	return fmt.Errorf("timeout waiting for %s", url)
}

func argvContainsFlag(argv []string, flag string) bool {
	for _, arg := range argv {
		if arg == flag {
			return true
		}
	}
	return false
}

func subcommandFlagInsertAt(argv []string) int {
	for i, arg := range argv {
		if arg == "machine" && i+1 < len(argv) {
			insertAt := i + 2
			if argv[i+1] == "restore" && insertAt < len(argv) && !strings.HasPrefix(argv[insertAt], "-") {
				insertAt++
			}
			for insertAt < len(argv) && strings.HasPrefix(argv[insertAt], "-") {
				insertAt++
				if insertAt < len(argv) && !strings.HasPrefix(argv[insertAt], "-") {
					insertAt++
				}
			}
			return insertAt
		}
	}
	return len(argv)
}

func insertRestoreArchive(argv []string, archive string) []string {
	for i, arg := range argv {
		if arg != "machine" || i+1 >= len(argv) || argv[i+1] != "restore" {
			continue
		}
		insertAt := i + 2
		if insertAt < len(argv) && !strings.HasPrefix(argv[insertAt], "-") {
			return argv
		}
		rest := append([]string{archive}, argv[insertAt:]...)
		return append(append([]string{}, argv[:insertAt]...), rest...)
	}
	return argv
}

func insertSubcommandFlags(argv []string, flags ...string) []string {
	insertAt := subcommandFlagInsertAt(argv)
	out := make([]string, 0, len(argv)+len(flags))
	out = append(out, argv[:insertAt]...)
	out = append(out, flags...)
	out = append(out, argv[insertAt:]...)
	return out
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

func ensureSessionDefaultArchive(t *testing.T, moduleRoot, serverBin, agentBin, cacheDir, token string) (string, error) {
	t.Helper()
	if err := os.MkdirAll(cacheDir, 0755); err != nil {
		return "", err
	}
	archive := filepath.Join(cacheDir, "default-prereq.tar.xz")
	lock := filepath.Join(cacheDir, "default-backup.lock")
	err := withFileLock(t, lock, func() error {
		if archiveHasXZMagicFile(archive) {
			return nil
		}
		seedHome, err := os.MkdirTemp("", "machine-backup-session-seed-*")
		if err != nil {
			return err
		}
		defer os.RemoveAll(seedHome)
		if err := seedServerHome(t, seedHome, false); err != nil {
			return err
		}
		credDir := filepath.Join(seedHome, ".ai-critic")
		if err := os.MkdirAll(credDir, 0755); err != nil {
			return err
		}
		credFile := filepath.Join(credDir, "server-credentials")
		if err := os.WriteFile(credFile, []byte(token+"\n"), 0600); err != nil {
			return err
		}
		agentHome, err := os.MkdirTemp("", "machine-backup-session-agent-*")
		if err != nil {
			return err
		}
		defer os.RemoveAll(agentHome)
		port := pickFreePort(27000 + portBaseFromTestName(DOCTEST_SESSION_ID)%500)
		serverURL := fmt.Sprintf("http://127.0.0.1:%d", port)
		killPort(port)
		serverCmd := exec.Command(serverBin, "--port", strconv.Itoa(port), "--credentials-file", credFile)
		serverCmd.Dir = seedHome
		serverCmd.Env = stripEnvPrefix(os.Environ(), "HOME=")
		serverCmd.Env = stripEnvPrefix(serverCmd.Env, lib.EnvAI_CRITIC_HOME+"=")
		serverCmd.Env = append(serverCmd.Env, "HOME="+seedHome, "AI_CRITIC_NO_OPEN_BROWSER=1")
		if err := serverCmd.Start(); err != nil {
			return fmt.Errorf("start session seed server: %w", err)
		}
		defer func() {
			if serverCmd.Process != nil {
				serverCmd.Process.Signal(syscall.SIGTERM)
				time.Sleep(100 * time.Millisecond)
				serverCmd.Process.Kill()
			}
		}()
		pingURL := fmt.Sprintf("http://127.0.0.1:%d/ping", port)
		if err := waitHTTPReady(pingURL, 30*time.Second); err != nil {
			return err
		}
		if err := verifyServerHome(t, serverURL, token, seedHome); err != nil {
			return err
		}
		agentEnv := stripEnvPrefix(os.Environ(), "HOME=")
		agentEnv = append(agentEnv, "HOME="+agentHome)
		backupArgs := []string{"--server", serverURL, "--token", token, "machine", "backup", "--output", archive}
		t.Logf("session default backup argv: %v", backupArgs)
		if code, out, errOut, runErr := runAgent(agentBin, backupArgs, agentEnv); runErr != nil {
			return runErr
		} else if code != 0 {
			return fmt.Errorf("session default backup exit %d:\n%s\n%s", code, out, errOut)
		}
		if !archiveHasXZMagicFile(archive) {
			return fmt.Errorf("session default archive missing xz magic: %s", archive)
		}
		t.Logf("session default archive written: %s", archive)
		return nil
	})
	if err != nil {
		return "", err
	}
	return archive, nil
}
```