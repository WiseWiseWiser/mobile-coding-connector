# Scenario

**Feature**: remote-agent machine backup and restore integration harness

```
# serverHome fixtures + server subprocess + isolated agent HOME
leaf Setup -> seed serverHome -> remote-agent machine backup|restore -> archive / stdout / server files
```

## Preconditions

1. Doctest injects `DOCTEST_SESSION_ID` (global in each generated test) to scope a
   file cache under `$TMPDIR/machine-backup-doctest-<session>/`
   (binaries built once; default prereq archive built once when reuse applies).
2. Session file locks (`flock`) serialize first-time cache population across parallel leaf packages.
3. Each leaf still gets an isolated `serverHome` / `agentHome`; only artifacts are shared.
4. Server runs with `HOME=serverHome` and cwd `serverHome` so backup scope matches fake machine home.

## Steps

1. Root `Run` builds binaries, seeds `serverHome`, starts server, writes agent config.
2. Leaf `Setup` narrows `Request` (flags, excludes, restore prereq backup, mutations).
3. `Run` may run a prereq `machine backup` before restore when `PrereqBackup` is set.
4. Leaf `Assert` checks exit code, CLI output, archive layout, and server home files.

## Context

Implements REQUIREMENT-DESIGN-remote-agent-machine-backup.md,
REQUIREMENT-DESIGN-machine-backup-exclusions.md,
REQUIREMENT-DESIGN-excluded-sizes.md,
REQUIREMENT-DESIGN-backup-large-dir-summary.md,
REQUIREMENT-DESIGN-large-dir-detail-deep.md, and
REQUIREMENT-DESIGN-backup-config-refinements.md, and
REQUIREMENT-DESIGN-machine-backup-git-repos.md, and
REQUIREMENT-DESIGN-git-repos-home-scan.md, and
REQUIREMENT-DESIGN-dry-run-meta-tables.md, and
REQUIREMENT-DESIGN-tailscale-config-meta.md, and
REQUIREMENT-DESIGN-systemd-services-meta.md, and
REQUIREMENT-DESIGN-cloudflared-config-meta.md. `SeedTailscaleMock` writes
`serverHome/bin/tailscale` (mock CLI), seeds bash/zsh history with tailscale
lines, and prepends `serverHome/bin` to the server subprocess `PATH`.
`SeedCloudflaredMock` writes `serverHome/bin/cloudflared` (mock CLI),
`serverHome/bin/pgrep` (stub that reads `.doctest-cloudflared.pid`),
`.doctest-cloudflared.pid` + `.doctest-cloudflared.cmdline` stubs, optional
`.cloudflared/config.yml` with fake credentials, bash history with cloudflared
quick-tunnel line, and prepends `serverHome/bin` to the server subprocess `PATH`.
`SeedSystemdMock` writes `serverHome/bin/systemctl` (mock CLI) and prepends
`serverHome/bin` to the server subprocess `PATH`; `SeedSystemdMockEmpty` sets
`SYSTEMD_MOCK_EMPTY=1` on the server subprocess so list-units returns `[]`.
Git fixture leaves skip when `git`
is not on PATH (`requireGit`). `SeedGitReposNonDot` seeds `projects/demo` via
`seedGitReposNonDotFixture`. `SeedGitReposOrigin` seeds `.wrk-test/main` and adds
`origin` remote `https://github.com/example/backup-fixture.git`. `SeedExcludedSizes` writes 1024 B /
512 B fixtures for per-rule EXCLUDED stats assertions. `SeedLargeDir` writes
`.big-test/` (>40 MB) and `.small-test/` for size-sorted DOT DIRS / LARGE SIZE
coverage. `SeedLargeDirDetailDeep` extends `SeedLargeDir` with `.deep-test/nested-big/`
(12 MB) and a small sibling; default `.cache` stays builtin-excluded. Flat
`LARGE DIR DETAIL` helpers parse `> <rel-path>  <size>` lines and assert
size-desc / path-asc sort. `SeedIncludedFetchSkills` writes small files under
paths removed from built-in exclusions.

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
	"regexp"
	"sort"
	"strconv"
	"strings"
	"syscall"
	"testing"
	"time"

	"github.com/xhd2015/ai-critic/script/lib"
)

func sessionCacheDir() string {
	return filepath.Join(os.TempDir(), "machine-backup-doctest-"+DOCTEST_SESSION_ID)
}

func withFileLock(t *testing.T, lockPath string, fn func() error) error {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(lockPath), 0755); err != nil {
		return err
	}
	f, err := os.OpenFile(lockPath, os.O_CREATE|os.O_RDWR, 0600)
	if err != nil {
		return err
	}
	defer f.Close()
	if err := syscall.Flock(int(f.Fd()), syscall.LOCK_EX); err != nil {
		return fmt.Errorf("flock %s: %w", lockPath, err)
	}
	defer syscall.Flock(int(f.Fd()), syscall.LOCK_UN)
	return fn()
}

func buildSessionBinariesOnce(t *testing.T, moduleRoot, cacheDir string) (serverBin, agentBin string) {
	t.Helper()
	if err := os.MkdirAll(cacheDir, 0755); err != nil {
		t.Fatal(err)
	}
	serverBin = filepath.Join(cacheDir, "ai-critic-server")
	agentBin = filepath.Join(cacheDir, "remote-agent")
	ready := filepath.Join(cacheDir, "binaries.ready")
	lock := filepath.Join(cacheDir, "build.lock")
	err := withFileLock(t, lock, func() error {
		if fileExists(ready) && fileExists(serverBin) && fileExists(agentBin) {
			return nil
		}
		for _, spec := range []struct {
			out string
			pkg string
		}{
			{serverBin, "."},
			{agentBin, "./cmd/remote-agent"},
		} {
			cmd := exec.Command("go", "build", "-o", spec.out, spec.pkg)
			cmd.Dir = moduleRoot
			out, err := cmd.CombinedOutput()
			if err != nil {
				return fmt.Errorf("build %s: %w\n%s", spec.pkg, err, string(out))
			}
		}
		return os.WriteFile(ready, []byte(time.Now().UTC().Format(time.RFC3339)), 0644)
	})
	if err != nil {
		t.Fatal(err)
	}
	t.Logf("session binaries cache: %s", cacheDir)
	return serverBin, agentBin
}

func needsCustomPrereqArchive(req *Request) bool {
	if len(req.ExcludePaths) > 0 || len(req.IncludePaths) > 0 {
		return true
	}
	return req.SeedDocker || req.SeedBackupMeta || req.SeedGitRepos || req.SeedGitReposWorktree || req.SeedGitReposEmpty || req.SeedGitReposOrigin || req.SeedTailscaleMock || req.SeedCloudflaredMock || req.SeedSystemdMock
}

func fileExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}

func archiveHasXZMagicFile(path string) bool {
	data, err := os.ReadFile(path)
	if err != nil || len(data) < 6 {
		return false
	}
	magic := []byte{0xFD, 0x37, 0x7A, 0x58, 0x5A, 0x00}
	return bytes.Equal(data[:6], magic)
}

const seededBackupMetaJSON = `{"seeded_meta":true,"marker":"pre-backup-old"}` + "\n"

type exclusionConfigJSON struct {
	Version           string `json:"version"`
	ExcludePaths      []struct {
		Path   string `json:"path"`
		Reason string `json:"reason"`
	} `json:"exclude_paths"`
	LargeDirThreshold string `json:"large_dir_threshold,omitempty"`
}

type userBackupConfigJSON struct {
	Version           string `json:"version"`
	ExcludePaths      []struct {
		Path   string `json:"path"`
		Reason string `json:"reason"`
	} `json:"exclude_paths"`
	LargeDirThreshold string `json:"large_dir_threshold,omitempty"`
}

type effectiveExclusionConfigJSON struct {
	Version           string `json:"version"`
	ExcludePaths      []struct {
		Path   string `json:"path"`
		Reason string `json:"reason"`
	} `json:"exclude_paths"`
	LargeDirThreshold string `json:"large_dir_threshold,omitempty"`
}

func userBackupConfigPath(home string) string {
	return filepath.Join(home, ".ai-critic", "backup-config.json")
}

func parseUserBackupConfigJSON(t *testing.T, raw []byte) userBackupConfigJSON {
	t.Helper()
	var cfg userBackupConfigJSON
	if err := json.Unmarshal(raw, &cfg); err != nil {
		t.Fatalf("parse user backup-config.json: %v\n%s", err, raw)
	}
	return cfg
}

func parseEffectiveExclusionConfigJSON(t *testing.T, raw []byte) effectiveExclusionConfigJSON {
	t.Helper()
	var cfg effectiveExclusionConfigJSON
	if err := json.Unmarshal(raw, &cfg); err != nil {
		t.Fatalf("parse effective exclusion config: %v\n%s", err, raw)
	}
	return cfg
}

func assertPersistedExcludeEmptyReason(t *testing.T, raw []byte, path string) {
	t.Helper()
	var doc map[string]json.RawMessage
	if err := json.Unmarshal(raw, &doc); err != nil {
		t.Fatalf("parse persisted config map: %v\n%s", err, raw)
	}
	entriesRaw, ok := doc["exclude_paths"]
	if !ok {
		t.Fatal("persisted exclude_paths missing")
	}
	var entries []map[string]json.RawMessage
	if err := json.Unmarshal(entriesRaw, &entries); err != nil {
		t.Fatalf("parse persisted exclude_paths: %v\n%s", err, entriesRaw)
	}
	found := false
	for _, entry := range entries {
		pathRaw, ok := entry["path"]
		if !ok {
			continue
		}
		var gotPath string
		if err := json.Unmarshal(pathRaw, &gotPath); err != nil {
			t.Fatalf("parse persisted path: %v", err)
		}
		if gotPath != path {
			continue
		}
		found = true
		if reasonRaw, ok := entry["reason"]; ok {
			var reason string
			if err := json.Unmarshal(reasonRaw, &reason); err != nil {
				t.Fatalf("parse persisted reason for %q: %v", path, err)
			}
			if strings.TrimSpace(reason) != "" {
				t.Fatalf("persisted exclude_paths[%q].reason = %q, want empty or omitted", path, reason)
			}
		}
	}
	if !found {
		t.Fatalf("persisted exclude_paths missing %q; raw:\n%s", path, raw)
	}
	if strings.Contains(string(raw), `"user excluded"`) {
		t.Fatalf("persisted config must not contain user excluded reason; raw:\n%s", raw)
	}
}

func assertEffectiveExcludeReason(t *testing.T, cfg effectiveExclusionConfigJSON, path, wantReason string) {
	t.Helper()
	for _, e := range cfg.ExcludePaths {
		if e.Path == path {
			if e.Reason != wantReason {
				t.Fatalf("effective exclude_paths[%q].reason = %q, want %q", path, e.Reason, wantReason)
			}
			return
		}
	}
	t.Fatalf("effective exclude_paths missing %q: %+v", path, cfg.ExcludePaths)
}

func appendPostPrereqSetConfigExcludes(t *testing.T, home string, entries []PostSetConfigExcludeEntry) error {
	t.Helper()
	cfgPath := userBackupConfigPath(home)
	raw, err := os.ReadFile(cfgPath)
	if err != nil {
		return fmt.Errorf("read %s: %w", cfgPath, err)
	}
	var doc struct {
		Version           string `json:"version"`
		ExcludePaths      []struct {
			Path   string `json:"path"`
			Reason string `json:"reason,omitempty"`
		} `json:"exclude_paths"`
		LargeDirThreshold string `json:"large_dir_threshold,omitempty"`
	}
	if err := json.Unmarshal(raw, &doc); err != nil {
		return fmt.Errorf("parse %s: %w", cfgPath, err)
	}
	existing := make(map[string]int, len(doc.ExcludePaths))
	for i, e := range doc.ExcludePaths {
		existing[e.Path] = i
	}
	for _, add := range entries {
		if idx, ok := existing[add.Path]; ok {
			doc.ExcludePaths[idx].Reason = add.Reason
			continue
		}
		doc.ExcludePaths = append(doc.ExcludePaths, struct {
			Path   string `json:"path"`
			Reason string `json:"reason,omitempty"`
		}{Path: add.Path, Reason: add.Reason})
	}
	updated, err := json.MarshalIndent(doc, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal %s: %w", cfgPath, err)
	}
	updated = append(updated, '\n')
	if err := os.WriteFile(cfgPath, updated, 0644); err != nil {
		return fmt.Errorf("write %s: %w", cfgPath, err)
	}
	return nil
}

func Setup(t *testing.T, req *Request) error {
	if req.Token == "" {
		req.Token = lib.TestPassword
	}
	return nil
}

func seedServerHome(t *testing.T, home string, withDocker bool) error {
	t.Helper()
	files := map[string]string{
		".bashrc":                    "export FAKE=1\n",
		".profile":                   "# profile\n",
		".ai-critic/ai-models.json":  `{"models":[]}` + "\n",
		".cargo/config.toml":         "[source.crates-io]\n",
		".cache/junk":                "cache data\n",
		".npm/x/package.json":        "{}\n",
		".cargo/registry/db/idx":     "registry\n",
		"Projects/visible.txt":       "not a dot entry\n",
	}
	for rel, content := range files {
		full := filepath.Join(home, filepath.FromSlash(rel))
		if err := os.MkdirAll(filepath.Dir(full), 0755); err != nil {
			return err
		}
		if err := os.WriteFile(full, []byte(content), 0644); err != nil {
			return err
		}
	}
	link := filepath.Join(home, ".local", "bin", "tool-link")
	if err := os.MkdirAll(filepath.Dir(link), 0755); err != nil {
		return err
	}
	if err := os.Symlink("../../.bashrc", link); err != nil {
		return err
	}
	if withDocker {
		dockerCfg := filepath.Join(home, ".docker", "config")
		if err := os.MkdirAll(filepath.Dir(dockerCfg), 0755); err != nil {
			return err
		}
		if err := os.WriteFile(dockerCfg, []byte(`{"auths":{}}`+"\n"), 0600); err != nil {
			return err
		}
	}
	seedExtendedExclusionFixtures(t, home)
	return nil
}

func elfStubBytes() []byte {
	// Minimal valid ELF64 LSB header (enough for magic-based detection).
	data := make([]byte, 104)
	copy(data, []byte{0x7f, 'E', 'L', 'F', 2, 1, 1, 0})
	data[18] = 0x3e // x86_64
	data[19] = 0x00
	return data
}

func sqliteStubBytes() []byte {
	data := make([]byte, 32)
	copy(data, []byte("SQLite format 3\x00"))
	return data
}

func jpegStubBytes() []byte {
	// Minimal JPEG SOI marker + padding (detected as image, not executable).
	data := make([]byte, 16)
	data[0] = 0xff
	data[1] = 0xd8
	data[2] = 0xff
	data[3] = 0xe0
	return data
}

func writeServerBinaryFile(t *testing.T, home, rel string, data []byte) {
	t.Helper()
	full := filepath.Join(home, filepath.FromSlash(rel))
	if err := os.MkdirAll(filepath.Dir(full), 0755); err != nil {
		t.Fatalf("mkdir %s: %v", filepath.Dir(full), err)
	}
	if err := os.WriteFile(full, data, 0755); err != nil {
		t.Fatalf("write %s: %v", full, err)
	}
}

func seedExtendedExclusionFixtures(t *testing.T, home string) {
	t.Helper()
	writeServerFile(t, home, ".ai-critic/config.json", `{"fixture":true}`+"\n")
	writeServerFile(t, home, ".ai-critic/service.log", "service log line\n")
	writeServerFile(t, home, ".ai-critic/keep.log", "keep this log line\n")
	writeServerBinaryFile(t, home, ".ai-critic/bin/stub", elfStubBytes())
	writeServerFile(t, home, ".live-and-love/upload-chunks/chunk-1", "chunk data\n")
	writeServerBinaryFile(t, home, ".local/share/opencode/opencode.db", sqliteStubBytes())
	writeServerBinaryFile(t, home, ".live-and-love/imgs/photo.jpg", jpegStubBytes())
	writeServerFile(t, home, ".codex/.tmp/junk", "codex tmp\n")
	writeServerFile(t, home, ".local/share/opencode/repos/foo/clone", "repo clone\n")
	writeServerFile(t, home, ".local/share/opencode/log/app.log", "opencode app log\n")
	writeServerFile(t, home, ".local/share/cursor-agent/versions/v1/pkg", "agent version\n")
	writeServerBinaryFile(t, home, ".opencode/bin/opencode", elfStubBytes())
	writeServerFile(t, home, ".config/confluence-fetch-skill/data/cache", "confluence cache\n")
}

const (
	largeDirChildABytes           = 30 * 1024 * 1024
	largeDirChildBBytes           = 20 * 1024 * 1024
	deepNestedBigBytes            = 12 * 1024 * 1024
	deepSmallSiblingBytes         = 1024
	defaultLargeDirThresholdBytes = 10 * 1024 * 1024
	largeDirDetailThresholdBytes  = 10 * 1024 * 1024
)

func seedLargeDirFixture(t *testing.T, home string) {
	t.Helper()
	writeServerBinaryFile(t, home, ".big-test/child-a", paddedBytes(largeDirChildABytes))
	writeServerBinaryFile(t, home, ".big-test/child-b", paddedBytes(largeDirChildBBytes))
}

func seedSmallDirForSortFixture(t *testing.T, home string) {
	t.Helper()
	writeServerFile(t, home, ".small-test/tiny.txt", "small fixture\n")
}

func seedLargeDirDetailDeepFixture(t *testing.T, home string) {
	t.Helper()
	seedLargeDirFixture(t, home)
	writeServerBinaryFile(t, home, ".deep-test/nested-big/file", paddedBytes(deepNestedBigBytes))
	writeServerBinaryFile(t, home, ".deep-test/small/tiny", paddedBytes(deepSmallSiblingBytes))
}

func seedIncludedFetchSkills(t *testing.T, home string) {
	t.Helper()
	writeServerFile(t, home, ".config/git-fetch-skill/data/cache", "git-fetch cache\n")
	writeServerFile(t, home, ".config/confluence-fetch-skill/data/note", "confluence note\n")
	writeServerFile(t, home, ".knowledge-index/agents.json", `{"agents":[]}`+"\n")
}

func seedKnowledgeHub(t *testing.T, home string) {
	t.Helper()
	writeServerFile(t, home, ".knowledge-hub/cache/item", "knowledge cache\n")
	writeServerFile(t, home, ".knowledge-index/idx/data", "index data\n")
}

const (
	gitFixtureCommitMsg       = "backup git fixture"
	gitNonDotFixtureCommitMsg = "non-dot fixture"
	gitFixtureOriginURL       = "https://github.com/example/backup-fixture.git"
)

const (
	tailscaleFixtureVersion   = "1.96.2"
	tailscaleFixtureSelfIP    = "100.64.209.66"
	tailscaleFixtureDNSName   = "samd-agent.example.ts.net"
	tailscaleFixturePeerAName = "peer-a"
	tailscaleFixturePeerAIP   = "100.69.30.59"
	tailscaleFixturePeerBName = "peer-b"
	tailscaleFixturePeerBIP   = "100.126.43.79"
	tailscaleFixtureSocks5    = "localhost:1055"
	tailscaleFixtureCmdline   = "tailscaled --tun=userspace-networking --socks5-server=localhost:1055"
)

const tailscaleMockScript = `#!/bin/sh
set -e
case "$1" in
version)
  if [ "$2" = "--json" ]; then
    printf '%s\n' '{"ClientVersion":"1.96.2","TUN":true}'
  else
    echo "1.96.2"
  fi
  ;;
status)
  if [ "$2" = "--json" ]; then
    cat <<'MOCK_EOF'
{
  "BackendState": "Running",
  "Version": "1.96.2",
  "Self": {
    "TailscaleIPs": ["100.64.209.66"],
    "DNSName": "samd-agent.example.ts.net"
  },
  "Peer": {
    "peerA123": {
      "DNSName": "peer-a",
      "TailscaleIPs": ["100.69.30.59"],
      "OS": "linux",
      "Online": false,
      "LastSeen": "2026-07-06T08:51:00Z"
    },
    "peerB456": {
      "DNSName": "peer-b",
      "TailscaleIPs": ["100.126.43.79"],
      "OS": "macOS",
      "Online": true
    }
  }
}
MOCK_EOF
  else
    echo "mock tailscale: status requires --json" >&2
    exit 1
  fi
  ;;
debug)
  if [ "$2" = "prefs" ]; then
    cat <<'MOCK_EOF'
{
  "PrivateNodeKey": "nodekey:fake-private-should-redact",
  "OldPrivateNodeKey": "nodekey:fake-old-should-redact",
  "NetworkLockKey": "nlkey:fake-lock-should-redact",
  "Config": {
    "PrivateNodeKey": "nodekey:fake-nested-should-redact"
  }
}
MOCK_EOF
  else
    echo "mock tailscale: unknown debug subcommand" >&2
    exit 1
  fi
  ;;
*)
  echo "mock tailscale: unsupported: $*" >&2
  exit 1
  ;;
esac
`

func prependPathToEnv(env []string, dir string) []string {
	for i, e := range env {
		if strings.HasPrefix(e, "PATH=") {
			env[i] = "PATH=" + dir + string(os.PathListSeparator) + strings.TrimPrefix(e, "PATH=")
			return env
		}
	}
	return append(env, "PATH="+dir)
}

func seedTailscaleMock(t *testing.T, home string) {
	t.Helper()
	binDir := filepath.Join(home, "bin")
	if err := os.MkdirAll(binDir, 0755); err != nil {
		t.Fatalf("mkdir %s: %v", binDir, err)
	}
	tailscaleBin := filepath.Join(binDir, "tailscale")
	if err := os.WriteFile(tailscaleBin, []byte(tailscaleMockScript), 0755); err != nil {
		t.Fatalf("write mock tailscale: %v", err)
	}
	writeServerFile(t, home, ".bash_history", "ls\ntailscale up\n")
	writeServerFile(t, home, ".zsh_history", "cd ~\ntailscaled --tun=userspace-networking --socks5-server=localhost:1055\n")
}

const (
	cloudflaredFixtureVersion     = "cloudflared 2026.1.2"
	cloudflaredFixturePID         = 4567
	cloudflaredFixtureURL         = "http://127.0.0.1:23712"
	cloudflaredFixtureHostname    = ""
	cloudflaredFixtureCmdline     = "cloudflared tunnel --url http://127.0.0.1:23712"
	cloudflaredFixtureTunnelError = "tunnel list requires cloudflare credentials"
	cloudflaredFixtureTunnelID    = "fake-tunnel-id-should-redact"
	cloudflaredFixtureCredFile    = "fake-tunnel-id.json"
)

const cloudflaredMockScript = `#!/bin/sh
set -e
case "$1" in
version)
  echo "` + cloudflaredFixtureVersion + `"
  ;;
tunnel)
  if [ "$2" = "list" ] && [ "$4" = "json" ]; then
    printf '%s\n' '[]'
    exit 0
  fi
  echo "mock cloudflared: unsupported tunnel subcommand: $*" >&2
  exit 1
  ;;
*)
  echo "mock cloudflared: unsupported: $*" >&2
  exit 1
  ;;
esac
`

const cloudflaredMockPgrepScript = `#!/bin/sh
set -e
for arg in "$@"; do
  case "$arg" in
  *cloudflared*)
    if [ -f "${HOME}/.doctest-cloudflared.pid" ]; then
      cat "${HOME}/.doctest-cloudflared.pid"
      exit 0
    fi
    ;;
  esac
done
if command -v /usr/bin/pgrep >/dev/null 2>&1; then
  exec /usr/bin/pgrep "$@"
fi
if command -v pgrep >/dev/null 2>&1; then
  exec pgrep "$@"
fi
exit 1
`

const cloudflaredFixtureConfigYAML = `tunnel: ` + cloudflaredFixtureTunnelID + `
credentials-file: ` + cloudflaredFixtureCredFile + `
ingress:
  - hostname: example.test
    service: http://localhost:8080
`

func seedCloudflaredMock(t *testing.T, home string) {
	t.Helper()
	binDir := filepath.Join(home, "bin")
	if err := os.MkdirAll(binDir, 0755); err != nil {
		t.Fatalf("mkdir %s: %v", binDir, err)
	}
	cloudflaredBin := filepath.Join(binDir, "cloudflared")
	if err := os.WriteFile(cloudflaredBin, []byte(cloudflaredMockScript), 0755); err != nil {
		t.Fatalf("write mock cloudflared: %v", err)
	}
	pgrepBin := filepath.Join(binDir, "pgrep")
	if err := os.WriteFile(pgrepBin, []byte(cloudflaredMockPgrepScript), 0755); err != nil {
		t.Fatalf("write mock pgrep: %v", err)
	}
	if err := os.WriteFile(filepath.Join(home, ".doctest-cloudflared.pid"), []byte(strconv.Itoa(cloudflaredFixturePID)+"\n"), 0644); err != nil {
		t.Fatalf("write .doctest-cloudflared.pid: %v", err)
	}
	if err := os.WriteFile(filepath.Join(home, ".doctest-cloudflared.cmdline"), []byte(cloudflaredFixtureCmdline), 0644); err != nil {
		t.Fatalf("write .doctest-cloudflared.cmdline: %v", err)
	}
	writeServerFile(t, home, ".cloudflared/config.yml", cloudflaredFixtureConfigYAML)
	writeServerFile(t, home, ".bash_history", "ls\n"+cloudflaredFixtureCmdline+"\n")
	writeServerFile(t, home, ".zsh_history", "cd ~\n")
}

type cloudflaredConfigSnapshot struct {
	Version     string `json:"version"`
	CapturedAt  string `json:"captured_at"`
	Running     bool   `json:"running"`
	VersionInfo struct {
		Text string `json:"text"`
	} `json:"version_info"`
	Process struct {
		PID     int    `json:"pid"`
		Cmdline string `json:"cmdline"`
	} `json:"process"`
	QuickTunnel struct {
		URL      string `json:"url"`
		Hostname string `json:"hostname"`
	} `json:"quick_tunnel"`
	Tunnels struct {
		Available bool            `json:"available"`
		Error     string          `json:"error,omitempty"`
		Items     []json.RawMessage `json:"items"`
	} `json:"tunnels"`
	Config struct {
		Path         string `json:"path"`
		Present      bool   `json:"present"`
		RedactedYAML string `json:"redacted_yaml"`
	} `json:"config"`
	Setup struct {
		BashHistory []string `json:"bash_history"`
		ZshHistory  []string `json:"zsh_history"`
	} `json:"setup"`
}

func parseCloudflaredConfigJSON(t *testing.T, raw []byte) cloudflaredConfigSnapshot {
	t.Helper()
	var snap cloudflaredConfigSnapshot
	if err := json.Unmarshal(raw, &snap); err != nil {
		t.Fatalf("parse cloudflared-config.json: %v\n%s", err, raw)
	}
	return snap
}

func assertCloudflaredConfigBasics(t *testing.T, snap cloudflaredConfigSnapshot) {
	t.Helper()
	if snap.Version != "1.0" {
		t.Fatalf("cloudflared snapshot version = %q, want 1.0", snap.Version)
	}
	if snap.CapturedAt == "" {
		t.Fatal("cloudflared snapshot missing captured_at")
	}
	if !snap.Running {
		t.Fatal("cloudflared snapshot running = false, want true")
	}
	if snap.VersionInfo.Text != cloudflaredFixtureVersion {
		t.Fatalf("cloudflared version_info.text = %q, want %q", snap.VersionInfo.Text, cloudflaredFixtureVersion)
	}
	if snap.Process.PID != cloudflaredFixturePID {
		t.Fatalf("cloudflared process.pid = %d, want %d", snap.Process.PID, cloudflaredFixturePID)
	}
	if snap.Process.Cmdline != cloudflaredFixtureCmdline {
		t.Fatalf("cloudflared process.cmdline = %q, want %q", snap.Process.Cmdline, cloudflaredFixtureCmdline)
	}
	if snap.QuickTunnel.URL != cloudflaredFixtureURL {
		t.Fatalf("cloudflared quick_tunnel.url = %q, want %q", snap.QuickTunnel.URL, cloudflaredFixtureURL)
	}
}

func assertCloudflaredConfigRedacted(t *testing.T, snap cloudflaredConfigSnapshot) {
	t.Helper()
	if !snap.Config.Present {
		t.Fatal("cloudflared config.present = false, want true")
	}
	yaml := snap.Config.RedactedYAML
	for _, forbidden := range []string{cloudflaredFixtureTunnelID, cloudflaredFixtureCredFile} {
		if strings.Contains(yaml, forbidden) {
			t.Fatalf("cloudflared config not redacted; still contains %q:\n%s", forbidden, yaml)
		}
	}
	if !strings.Contains(strings.ToLower(yaml), "redact") {
		t.Fatalf("cloudflared config redacted_yaml missing redaction marker; got:\n%s", yaml)
	}
}

func assertCloudflaredSetupHistory(t *testing.T, snap cloudflaredConfigSnapshot) {
	t.Helper()
	var bashFound bool
	for _, line := range snap.Setup.BashHistory {
		if strings.Contains(strings.ToLower(line), "cloudflared") {
			bashFound = true
		}
	}
	if !bashFound {
		t.Fatalf("cloudflared setup.bash_history missing cloudflared line; got %v", snap.Setup.BashHistory)
	}
}

const (
	systemdFixtureUserUnit        = "agent-proxy.service"
	systemdFixtureUserPID         = 4521
	systemdFixtureUserDesc        = "AI Critic remote agent proxy"
	systemdFixtureUserUnitRel     = ".config/systemd/user/agent-proxy.service"
	systemdFixtureTailscaledUnit  = "tailscaled.service"
	systemdFixtureTailscaledPID   = 1234
	systemdFixtureTailscaledDesc  = "Tailscale node agent"
	systemdFixtureTailscaledRel   = "lib/systemd/system/tailscaled.service"
	systemdFixtureDockerUnit      = "docker.service"
	systemdFixtureDockerPID       = 890
	systemdFixtureDockerDesc      = "Docker Application Container Engine"
	systemdFixtureDockerRel       = "lib/systemd/system/docker.service"
)

const systemdMockScript = `#!/bin/sh
set -e

if [ "$1" = "--version" ]; then
  echo "systemd 252 (252-16.el9)"
  exit 0
fi

empty="${SYSTEMD_MOCK_EMPTY:-0}"
user_scope=0
for arg in "$@"; do
  if [ "$arg" = "--user" ]; then
    user_scope=1
    break
  fi
done

case "$*" in
*list-units*--type=service*--state=running*--output=json*)
  if [ "$empty" = "1" ]; then
    printf '%s\n' '[]'
    exit 0
  fi
  if [ "$user_scope" = "1" ]; then
    cat <<'MOCK_EOF'
[{"unit":"agent-proxy.service","load":"loaded","active":"active","sub":"running","description":"AI Critic remote agent proxy"}]
MOCK_EOF
  else
    cat <<'MOCK_EOF'
[{"unit":"tailscaled.service","load":"loaded","active":"active","sub":"running","description":"Tailscale node agent"},{"unit":"docker.service","load":"loaded","active":"active","sub":"running","description":"Docker Application Container Engine"}]
MOCK_EOF
  fi
  exit 0
  ;;
esac

unit=""
if [ "$1" = "--user" ] && [ "$2" = "show" ]; then
  unit="$3"
elif [ "$1" = "show" ]; then
  unit="$2"
fi

if [ -n "$unit" ]; then
  case "$unit" in
  agent-proxy.service)
    printf 'Description=%s\nMainPID=%d\nFragmentPath=%s/%s\n' \
      "AI Critic remote agent proxy" 4521 "${HOME}" ".config/systemd/user/agent-proxy.service"
    ;;
  tailscaled.service)
    printf 'Description=%s\nMainPID=%d\nFragmentPath=%s/%s\n' \
      "Tailscale node agent" 1234 "${HOME}" "lib/systemd/system/tailscaled.service"
    ;;
  docker.service)
    printf 'Description=%s\nMainPID=%d\nFragmentPath=%s/%s\n' \
      "Docker Application Container Engine" 890 "${HOME}" "lib/systemd/system/docker.service"
    ;;
  *)
    echo "mock systemctl: unknown unit $unit" >&2
    exit 1
    ;;
  esac
  exit 0
fi

echo "mock systemctl: unsupported: $*" >&2
exit 1
`

func seedSystemdMock(t *testing.T, home string) {
	t.Helper()
	binDir := filepath.Join(home, "bin")
	if err := os.MkdirAll(binDir, 0755); err != nil {
		t.Fatalf("mkdir %s: %v", binDir, err)
	}
	systemctlBin := filepath.Join(binDir, "systemctl")
	if err := os.WriteFile(systemctlBin, []byte(systemdMockScript), 0755); err != nil {
		t.Fatalf("write mock systemctl: %v", err)
	}
}

type systemdUnitSnapshot struct {
	Unit        string `json:"unit"`
	Load        string `json:"load"`
	Active      string `json:"active"`
	Sub         string `json:"sub"`
	Description string `json:"description"`
	MainPID     int    `json:"main_pid"`
	UnitFile    string `json:"unit_file"`
}

type systemdScopeSnapshot struct {
	Available    bool                  `json:"available"`
	RunningCount int                   `json:"running_count"`
	Error        string                `json:"error,omitempty"`
	Units        []systemdUnitSnapshot `json:"units"`
}

type systemdServicesSnapshot struct {
	Version          string `json:"version"`
	CapturedAt       string `json:"captured_at"`
	SystemdAvailable bool   `json:"systemd_available"`
	Scopes           struct {
		User   systemdScopeSnapshot `json:"user"`
		System systemdScopeSnapshot `json:"system"`
	} `json:"scopes"`
}

func parseSystemdServicesJSON(t *testing.T, raw []byte) systemdServicesSnapshot {
	t.Helper()
	var snap systemdServicesSnapshot
	if err := json.Unmarshal(raw, &snap); err != nil {
		t.Fatalf("parse systemd-services.json: %v\n%s", err, raw)
	}
	return snap
}

func assertSystemdServicesBasics(t *testing.T, snap systemdServicesSnapshot) {
	t.Helper()
	if snap.Version != "1.0" {
		t.Fatalf("systemd snapshot version = %q, want 1.0", snap.Version)
	}
	if snap.CapturedAt == "" {
		t.Fatal("systemd snapshot missing captured_at")
	}
	if !snap.SystemdAvailable {
		t.Fatal("systemd snapshot systemd_available = false, want true")
	}
}

func assertSystemdServicesRunningCounts(t *testing.T, snap systemdServicesSnapshot, userCount, systemCount int) {
	t.Helper()
	if snap.Scopes.User.RunningCount != userCount {
		t.Fatalf("systemd user running_count = %d, want %d", snap.Scopes.User.RunningCount, userCount)
	}
	if snap.Scopes.System.RunningCount != systemCount {
		t.Fatalf("systemd system running_count = %d, want %d", snap.Scopes.System.RunningCount, systemCount)
	}
}

func assertSystemdServicesHasUnit(t *testing.T, snap systemdServicesSnapshot, scope, unit string, wantPID int, wantDesc string) {
	t.Helper()
	var units []systemdUnitSnapshot
	switch scope {
	case "user":
		units = snap.Scopes.User.Units
	case "system":
		units = snap.Scopes.System.Units
	default:
		t.Fatalf("unknown systemd scope %q", scope)
	}
	for _, u := range units {
		if u.Unit != unit {
			continue
		}
		if u.MainPID != wantPID {
			t.Fatalf("systemd %s unit %q main_pid = %d, want %d", scope, unit, u.MainPID, wantPID)
		}
		if u.Description != wantDesc {
			t.Fatalf("systemd %s unit %q description = %q, want %q", scope, unit, u.Description, wantDesc)
		}
		if u.Load != "loaded" || u.Active != "active" || u.Sub != "running" {
			t.Fatalf("systemd %s unit %q state = load:%s active:%s sub:%s, want loaded/active/running",
				scope, unit, u.Load, u.Active, u.Sub)
		}
		if u.UnitFile == "" {
			t.Fatalf("systemd %s unit %q missing unit_file", scope, unit)
		}
		return
	}
	t.Fatalf("systemd %s scope missing unit %q; units=%v", scope, unit, units)
}

func assertSystemdServicesTableHeaders(t *testing.T, section string) {
	t.Helper()
	for _, h := range []string{"UNIT", "PID", "DESCRIPTION"} {
		if !strings.Contains(section, h) {
			t.Fatalf("SYSTEMD SERVICES table missing %q column header; section:\n%s", h, section)
		}
	}
}

type tailscaleConfigSnapshot struct {
	Version     string `json:"version"`
	CapturedAt  string `json:"captured_at"`
	Running     bool   `json:"running"`
	VersionInfo struct {
		Text string          `json:"text"`
		JSON json.RawMessage `json:"json"`
	} `json:"version_info"`
	Daemon struct {
		PID                 int    `json:"pid"`
		Cmdline             string `json:"cmdline"`
		StatePath           string `json:"state_path"`
		SocketPath          string `json:"socket_path"`
		UserspaceNetworking bool   `json:"userspace_networking"`
		Socks5Server        string `json:"socks5_server"`
	} `json:"daemon"`
	Status json.RawMessage `json:"status"`
	Prefs  json.RawMessage `json:"prefs"`
	Setup  struct {
		Summary     string   `json:"summary"`
		Steps       []string `json:"steps"`
		Commands    []string `json:"commands"`
		BashHistory []string `json:"bash_history"`
		ZshHistory  []string `json:"zsh_history"`
		Notes       []string `json:"notes"`
	} `json:"setup"`
}

func parseTailscaleConfigJSON(t *testing.T, raw []byte) tailscaleConfigSnapshot {
	t.Helper()
	var snap tailscaleConfigSnapshot
	if err := json.Unmarshal(raw, &snap); err != nil {
		t.Fatalf("parse tailscale-config.json: %v\n%s", err, raw)
	}
	return snap
}

func assertTailscaleConfigBasics(t *testing.T, snap tailscaleConfigSnapshot) {
	t.Helper()
	if snap.Version != "1.0" {
		t.Fatalf("tailscale snapshot version = %q, want 1.0", snap.Version)
	}
	if snap.CapturedAt == "" {
		t.Fatal("tailscale snapshot missing captured_at")
	}
	if !snap.Running {
		t.Fatal("tailscale snapshot running = false, want true")
	}
	if snap.VersionInfo.Text != tailscaleFixtureVersion {
		t.Fatalf("tailscale version_info.text = %q, want %q", snap.VersionInfo.Text, tailscaleFixtureVersion)
	}
}

func assertTailscalePrefsRedacted(t *testing.T, prefsRaw json.RawMessage) {
	t.Helper()
	prefsStr := string(prefsRaw)
	for _, forbidden := range []string{
		"nodekey:fake-private-should-redact",
		"nodekey:fake-old-should-redact",
		"nlkey:fake-lock-should-redact",
		"nodekey:fake-nested-should-redact",
	} {
		if strings.Contains(prefsStr, forbidden) {
			t.Fatalf("tailscale prefs not redacted; still contains %q:\n%s", forbidden, prefsStr)
		}
	}
	var prefs map[string]json.RawMessage
	if err := json.Unmarshal(prefsRaw, &prefs); err != nil {
		t.Fatalf("parse tailscale prefs: %v\n%s", err, prefsRaw)
	}
	for _, key := range []string{"PrivateNodeKey", "OldPrivateNodeKey", "NetworkLockKey"} {
		if val, ok := prefs[key]; ok {
			var s string
			if err := json.Unmarshal(val, &s); err == nil && s != "" && !strings.Contains(strings.ToLower(s), "redact") {
				t.Fatalf("tailscale prefs[%q] = %q, want redacted or omitted", key, s)
			}
		}
	}
}

func assertTailscaleSetupHistory(t *testing.T, snap tailscaleConfigSnapshot) {
	t.Helper()
	var bashFound, zshFound bool
	for _, line := range snap.Setup.BashHistory {
		if strings.Contains(strings.ToLower(line), "tailscale") {
			bashFound = true
		}
	}
	for _, line := range snap.Setup.ZshHistory {
		if strings.Contains(strings.ToLower(line), "tailscale") {
			zshFound = true
		}
	}
	if !bashFound {
		t.Fatalf("tailscale setup.bash_history missing tailscale line; got %v", snap.Setup.BashHistory)
	}
	if !zshFound {
		t.Fatalf("tailscale setup.zsh_history missing tailscale line; got %v", snap.Setup.ZshHistory)
	}
}

func requireGit(t *testing.T) {
	t.Helper()
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git not in PATH")
	}
}

func gitRun(t *testing.T, dir string, args ...string) {
	t.Helper()
	requireGit(t)
	cmd := exec.Command("git", args...)
	cmd.Dir = dir
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("git %v in %s: %v\n%s", args, dir, err, out)
	}
}

func gitInitRepo(t *testing.T, dir string) {
	t.Helper()
	requireGit(t)
	if err := os.MkdirAll(dir, 0755); err != nil {
		t.Fatalf("mkdir %s: %v", dir, err)
	}
	gitRun(t, dir, "init")
	gitRun(t, dir, "config", "user.email", "test@example.com")
	gitRun(t, dir, "config", "user.name", "Test User")
	gitRun(t, dir, "branch", "-M", "main")
	readme := filepath.Join(dir, "README.md")
	if err := os.WriteFile(readme, []byte("seed\n"), 0644); err != nil {
		t.Fatalf("write README.md: %v", err)
	}
	gitRun(t, dir, "add", "README.md")
	gitRun(t, dir, "commit", "-m", "Initial commit")
}

func gitWorktreeAdd(t *testing.T, mainDir, wtDir, branch string) {
	t.Helper()
	requireGit(t)
	if err := os.MkdirAll(filepath.Dir(wtDir), 0755); err != nil {
		t.Fatalf("mkdir worktree parent: %v", err)
	}
	gitRun(t, mainDir, "worktree", "add", "-b", branch, wtDir)
}

func seedGitReposNonDotFixture(t *testing.T, home string) {
	t.Helper()
	requireGit(t)
	demoDir := filepath.Join(home, "projects", "demo")
	if err := os.MkdirAll(demoDir, 0755); err != nil {
		t.Fatalf("mkdir %s: %v", demoDir, err)
	}
	gitRun(t, demoDir, "init")
	gitRun(t, demoDir, "config", "user.email", "test@example.com")
	gitRun(t, demoDir, "config", "user.name", "Test User")
	gitRun(t, demoDir, "branch", "-M", "main")
	readme := filepath.Join(demoDir, "README.md")
	if err := os.WriteFile(readme, []byte("non-dot fixture\n"), 0644); err != nil {
		t.Fatalf("write README.md: %v", err)
	}
	gitRun(t, demoDir, "add", "README.md")
	gitRun(t, demoDir, "commit", "-m", gitNonDotFixtureCommitMsg)
}

func seedGitReposOriginFixture(t *testing.T, home string) {
	t.Helper()
	requireGit(t)
	seedGitReposFixture(t, home)
	mainDir := filepath.Join(home, ".wrk-test", "main")
	gitRun(t, mainDir, "remote", "add", "origin", gitFixtureOriginURL)
}

func seedGitReposFixture(t *testing.T, home string) {
	t.Helper()
	requireGit(t)
	mainDir := filepath.Join(home, ".wrk-test", "main")
	if err := os.MkdirAll(mainDir, 0755); err != nil {
		t.Fatalf("mkdir %s: %v", mainDir, err)
	}
	gitRun(t, mainDir, "init")
	gitRun(t, mainDir, "config", "user.email", "test@example.com")
	gitRun(t, mainDir, "config", "user.name", "Test User")
	gitRun(t, mainDir, "branch", "-M", "main")
	readme := filepath.Join(mainDir, "README.md")
	if err := os.WriteFile(readme, []byte("fixture\n"), 0644); err != nil {
		t.Fatalf("write README.md: %v", err)
	}
	gitRun(t, mainDir, "add", "README.md")
	gitRun(t, mainDir, "commit", "-m", gitFixtureCommitMsg)
}

func seedGitReposEmptyFixture(t *testing.T, home string) {
	t.Helper()
	requireGit(t)
	emptyDir := filepath.Join(home, ".wrk-test", "empty")
	if err := os.MkdirAll(emptyDir, 0755); err != nil {
		t.Fatalf("mkdir %s: %v", emptyDir, err)
	}
	gitRun(t, emptyDir, "init")
	gitRun(t, emptyDir, "config", "user.email", "test@example.com")
	gitRun(t, emptyDir, "config", "user.name", "Test User")
	gitRun(t, emptyDir, "branch", "-M", "main")
}

func seedGitReposWorktreeFixture(t *testing.T, home string) {
	t.Helper()
	requireGit(t)
	seedGitReposFixture(t, home)
	mainDir := filepath.Join(home, ".wrk-test", "main")
	wtDir := filepath.Join(home, ".wrk-test", "feature-wt")
	gitWorktreeAdd(t, mainDir, wtDir, "feature/foo")
	readme := filepath.Join(wtDir, "README.md")
	data, err := os.ReadFile(readme)
	if err != nil {
		t.Fatalf("read worktree README.md: %v", err)
	}
	if err := os.WriteFile(readme, append(data, []byte("dirty line\n")...), 0644); err != nil {
		t.Fatalf("dirty worktree README.md: %v", err)
	}
}

func seedGitReposMaxDepthFixture(t *testing.T, home string) {
	t.Helper()
	requireGit(t)
	deepDir := filepath.Join(home, ".wrk-test", "a", "b", "c", "deep-repo")
	if err := os.MkdirAll(deepDir, 0755); err != nil {
		t.Fatalf("mkdir %s: %v", deepDir, err)
	}
	gitRun(t, deepDir, "init")
	gitRun(t, deepDir, "config", "user.email", "test@example.com")
	gitRun(t, deepDir, "config", "user.name", "Test User")
	gitRun(t, deepDir, "branch", "-M", "main")
	readme := filepath.Join(deepDir, "README.md")
	if err := os.WriteFile(readme, []byte("deep\n"), 0644); err != nil {
		t.Fatalf("write deep README.md: %v", err)
	}
	gitRun(t, deepDir, "add", "README.md")
	gitRun(t, deepDir, "commit", "-m", "deep fixture")
}

var gitShortSHARE = regexp.MustCompile(`\b[0-9a-f]{7}\b`)

func dryRunSummaryRest(combined string) string {
	idxPlan := strings.Index(combined, "dry-run: machine backup plan")
	if idxPlan < 0 {
		return ""
	}
	return combined[idxPlan:]
}

func gitReposSummarySection(combined string) string {
	rest := dryRunSummaryRest(combined)
	if rest == "" {
		return ""
	}
	idxGit := strings.Index(rest, "GIT REPOS")
	if idxGit < 0 {
		return ""
	}
	rest = rest[idxGit:]
	if idxInstalled := strings.Index(rest, "\n  INSTALLED SOFTWARE"); idxInstalled >= 0 {
		return rest[:idxInstalled]
	}
	if idxTotal := strings.Index(rest, "\n  TOTAL:"); idxTotal >= 0 {
		return rest[:idxTotal]
	}
	return rest
}

func installedSummarySection(combined string) string {
	rest := dryRunSummaryRest(combined)
	if rest == "" {
		return ""
	}
	idx := strings.Index(rest, "INSTALLED SOFTWARE")
	if idx < 0 {
		return ""
	}
	rest = rest[idx:]
	if idxEnv := strings.Index(rest, "\n  ENV(.backup/ENV):"); idxEnv >= 0 {
		return rest[:idxEnv]
	}
	if idxTotal := strings.Index(rest, "\n  TOTAL:"); idxTotal >= 0 {
		return rest[:idxTotal]
	}
	return rest
}

func envSummarySection(combined string) string {
	rest := dryRunSummaryRest(combined)
	if rest == "" {
		return ""
	}
	idx := strings.Index(rest, "ENV(.backup/ENV):")
	if idx < 0 {
		return ""
	}
	rest = rest[idx:]
	if idxTail := strings.Index(rest, "\n  TAILSCALE(.backup/tailscale-config.json):"); idxTail >= 0 {
		return rest[:idxTail]
	}
	if idxCloud := strings.Index(rest, "\n  CLOUDFLARED(.backup/cloudflared-config.json):"); idxCloud >= 0 {
		return rest[:idxCloud]
	}
	if idxSystemd := strings.Index(rest, "\n  SYSTEMD SERVICES(.backup/systemd-services.json):"); idxSystemd >= 0 {
		return rest[:idxSystemd]
	}
	if idxTotal := strings.Index(rest, "\n  TOTAL:"); idxTotal >= 0 {
		return rest[:idxTotal]
	}
	return rest
}

// metaSectionHeaderLines returns the first n lines of a dry-run meta section for
// assert.Output v2 prefix checks (tool/env rows may follow). Appends a trailing
// newline so actual matches v2 templates authored with a closing blank line.
func metaSectionHeaderLines(section string, n int) string {
	if section == "" || n <= 0 {
		return section
	}
	lines := strings.Split(strings.TrimSuffix(section, "\n"), "\n")
	if len(lines) <= n {
		if strings.HasSuffix(section, "\n") {
			return section
		}
		return section + "\n"
	}
	return strings.Join(lines[:n], "\n") + "\n"
}

func tailscaleSummarySection(combined string) string {
	rest := dryRunSummaryRest(combined)
	if rest == "" {
		return ""
	}
	idx := strings.Index(rest, "TAILSCALE(.backup/tailscale-config.json):")
	if idx < 0 {
		return ""
	}
	rest = rest[idx:]
	if idxCloud := strings.Index(rest, "\n  CLOUDFLARED(.backup/cloudflared-config.json):"); idxCloud >= 0 {
		return rest[:idxCloud]
	}
	if idxSystemd := strings.Index(rest, "\n  SYSTEMD SERVICES(.backup/systemd-services.json):"); idxSystemd >= 0 {
		return rest[:idxSystemd]
	}
	if idxTotal := strings.Index(rest, "\n  TOTAL:"); idxTotal >= 0 {
		return rest[:idxTotal]
	}
	return rest
}

func cloudflaredSummarySection(combined string) string {
	rest := dryRunSummaryRest(combined)
	if rest == "" {
		return ""
	}
	idx := strings.Index(rest, "CLOUDFLARED(.backup/cloudflared-config.json):")
	if idx < 0 {
		return ""
	}
	rest = rest[idx:]
	if idxSystemd := strings.Index(rest, "\n  SYSTEMD SERVICES(.backup/systemd-services.json):"); idxSystemd >= 0 {
		return rest[:idxSystemd]
	}
	if idxTotal := strings.Index(rest, "\n  TOTAL:"); idxTotal >= 0 {
		return rest[:idxTotal]
	}
	return rest
}

func systemdServicesSummarySection(combined string) string {
	rest := dryRunSummaryRest(combined)
	if rest == "" {
		return ""
	}
	idx := strings.Index(rest, "SYSTEMD SERVICES(.backup/systemd-services.json):")
	if idx < 0 {
		return ""
	}
	rest = rest[idx:]
	if idxTotal := strings.Index(rest, "\n  TOTAL:"); idxTotal >= 0 {
		return rest[:idxTotal]
	}
	return rest
}

// metaSummarySection spans GIT REPOS through ENV (exclusive of TOTAL).
func metaSummarySection(combined string) string {
	rest := dryRunSummaryRest(combined)
	if rest == "" {
		return ""
	}
	idxGit := strings.Index(rest, "GIT REPOS")
	if idxGit < 0 {
		idxGit = strings.Index(rest, "INSTALLED SOFTWARE")
	}
	if idxGit < 0 {
		return ""
	}
	rest = rest[idxGit:]
	if idxTotal := strings.Index(rest, "\n  TOTAL:"); idxTotal >= 0 {
		return rest[:idxTotal]
	}
	return rest
}

func assertGitReposTableHeaders(t *testing.T, section string) {
	t.Helper()
	for _, h := range []string{"KIND", "PATH", "BRANCH", "SHA", "STATUS", "ORIGIN", "MESSAGE"} {
		if !strings.Contains(section, h) {
			t.Fatalf("GIT REPOS table missing %q column header; section:\n%s", h, section)
		}
	}
}

func assertInstalledTableHeaders(t *testing.T, section string) {
	t.Helper()
	for _, h := range []string{"NAME", "VERSION", "PATH"} {
		if !strings.Contains(section, h) {
			t.Fatalf("INSTALLED SOFTWARE table missing %q column header; section:\n%s", h, section)
		}
	}
}

func assertMetaSectionsBeforeTotal(t *testing.T, combined string) {
	t.Helper()
	rest := dryRunSummaryRest(combined)
	if rest == "" {
		t.Fatalf("missing dry-run summary block; got:\n%s", combined)
	}
	gitIdx := strings.Index(rest, "GIT REPOS")
	installedIdx := strings.Index(rest, "INSTALLED SOFTWARE")
	envIdx := strings.Index(rest, "ENV(.backup/ENV):")
	tailscaleIdx := strings.Index(rest, "TAILSCALE(.backup/tailscale-config.json):")
	cloudflaredIdx := strings.Index(rest, "CLOUDFLARED(.backup/cloudflared-config.json):")
	systemdIdx := strings.Index(rest, "SYSTEMD SERVICES(.backup/systemd-services.json):")
	totalIdx := strings.Index(rest, "TOTAL:")
	if gitIdx < 0 || installedIdx < 0 || envIdx < 0 || totalIdx < 0 {
		t.Fatalf("missing meta section markers; git=%d installed=%d env=%d total=%d\n%s",
			gitIdx, installedIdx, envIdx, totalIdx, rest)
	}
	if !(gitIdx < installedIdx && installedIdx < envIdx && envIdx < totalIdx) {
		t.Fatalf("meta sections out of order (want GIT REPOS → INSTALLED → ENV → [TAILSCALE?] → [CLOUDFLARED?] → [SYSTEMD?] → TOTAL); indices git=%d installed=%d env=%d total=%d",
			gitIdx, installedIdx, envIdx, totalIdx)
	}
	prevIdx := envIdx
	if tailscaleIdx >= 0 {
		if !(prevIdx < tailscaleIdx && tailscaleIdx < totalIdx) {
			t.Fatalf("TAILSCALE section out of order (want after ENV, before TOTAL); prev=%d tailscale=%d total=%d",
				prevIdx, tailscaleIdx, totalIdx)
		}
		prevIdx = tailscaleIdx
	}
	if cloudflaredIdx >= 0 {
		if !(prevIdx < cloudflaredIdx && cloudflaredIdx < totalIdx) {
			t.Fatalf("CLOUDFLARED section out of order (want after ENV/[TAILSCALE?], before TOTAL); prev=%d cloudflared=%d total=%d",
				prevIdx, cloudflaredIdx, totalIdx)
		}
		prevIdx = cloudflaredIdx
	}
	if systemdIdx >= 0 {
		if !(prevIdx < systemdIdx && systemdIdx < totalIdx) {
			t.Fatalf("SYSTEMD SERVICES section out of order (want after ENV/[TAILSCALE?]/[CLOUDFLARED?], before TOTAL); prev=%d systemd=%d total=%d",
				prevIdx, systemdIdx, totalIdx)
		}
	}
}

func assertGitReposSummaryContains(t *testing.T, combined string, needles ...string) {
	t.Helper()
	section := gitReposSummarySection(combined)
	if section == "" {
		t.Fatalf("missing GIT REPOS summary section; got:\n%s", combined)
	}
	for _, n := range needles {
		if !strings.Contains(section, n) {
			t.Fatalf("GIT REPOS section missing %q; section:\n%s", n, section)
		}
	}
}

func assertStdoutEndsWithNewline(t *testing.T, stdout string) {
	t.Helper()
	if stdout == "" {
		t.Fatal("stdout empty; want trailing newline")
	}
	if !strings.HasSuffix(stdout, "\n") {
		tail := 40
		if len(stdout) < tail {
			tail = len(stdout)
		}
		t.Fatalf("stdout missing trailing newline; ends with %q", stdout[len(stdout)-tail:])
	}
}

type gitRepoWorktreesSnapshot struct {
	Version    string `json:"version"`
	CapturedAt string `json:"captured_at"`
	Repos      []struct {
		Path      string `json:"path"`
		Branch    string `json:"branch"`
		CommitSHA string `json:"commit_sha"`
		CommitMsg string `json:"commit_msg"`
		Status    string `json:"status"`
		OriginURL string `json:"origin_url,omitempty"`
		Worktrees []struct {
			Path      string `json:"path"`
			Branch    string `json:"branch"`
			CommitSHA string `json:"commit_sha"`
			CommitMsg string `json:"commit_msg"`
			Status    string `json:"status"`
			OriginURL string `json:"origin_url,omitempty"`
		} `json:"worktrees,omitempty"`
	} `json:"repos"`
}

func parseGitRepoWorktreesJSON(t *testing.T, raw []byte) gitRepoWorktreesSnapshot {
	t.Helper()
	var snap gitRepoWorktreesSnapshot
	if err := json.Unmarshal(raw, &snap); err != nil {
		t.Fatalf("parse git-repo-worktrees.json: %v\n%s", err, raw)
	}
	return snap
}

func assertGitRepoSnapshotBasics(t *testing.T, snap gitRepoWorktreesSnapshot, repoPath string) {
	t.Helper()
	if snap.Version != "1.0" {
		t.Fatalf("git snapshot version = %q, want 1.0", snap.Version)
	}
	if snap.CapturedAt == "" {
		t.Fatal("git snapshot missing captured_at")
	}
	var found bool
	for _, repo := range snap.Repos {
		if repo.Path != repoPath {
			continue
		}
		found = true
		if len(repo.CommitSHA) != 7 || !gitShortSHARE.MatchString(repo.CommitSHA) {
			t.Fatalf("repo %q commit_sha = %q, want 7-char hex short sha", repoPath, repo.CommitSHA)
		}
		if repo.Status == "" {
			t.Fatalf("repo %q missing status", repoPath)
		}
	}
	if !found {
		t.Fatalf("git snapshot missing repo path %q; repos=%v", repoPath, snap.Repos)
	}
}

func assertGitRepoSnapshotOriginURL(t *testing.T, snap gitRepoWorktreesSnapshot, repoPath, want string) {
	t.Helper()
	for _, repo := range snap.Repos {
		if repo.Path != repoPath {
			continue
		}
		if want == "" {
			if repo.OriginURL != "" {
				t.Fatalf("repo %q origin_url = %q, want omitted", repoPath, repo.OriginURL)
			}
			return
		}
		if repo.OriginURL != want {
			t.Fatalf("repo %q origin_url = %q, want %q", repoPath, repo.OriginURL, want)
		}
		return
	}
	t.Fatalf("git snapshot missing repo path %q", repoPath)
}

func exclusionConfigHasPath(cfg exclusionConfigJSON, path string) (reason string, ok bool) {
	for _, e := range cfg.ExcludePaths {
		if e.Path == path {
			return e.Reason, true
		}
	}
	return "", false
}

func assertExclusionConfigV11(t *testing.T, cfg exclusionConfigJSON) {
	t.Helper()
	if cfg.Version != "1.1" {
		t.Fatalf("version = %q, want 1.1", cfg.Version)
	}
	wantEntries := map[string]string{
		"**(binary)":                          "executable binaries (reinstallable)",
		"**/*.log":                            "log files",
		"**/upload-chunks":                    "incomplete upload temp state",
		".local/share/cursor-agent/versions":  "Cursor agent version cache",
		".opencode/bin":                       "OpenCode binary (reinstallable)",
		".codex/.tmp":                         "Codex temporary plugin cache",
		".codex/skills/.system":               "Codex system skills cache",
		".local/share/opencode/repos":         "OpenCode repo clone cache",
		".local/share/opencode/snapshot":      "OpenCode snapshot cache",
		".local/share/opencode/log":           "OpenCode application logs",
		".grok/marketplace-cache":             "Grok plugin marketplace git cache",
		".grok/vendor":                        "Grok vendored dependencies cache",
		".grok/logs":                          "Grok application logs",
		".cache":                              "temporary application cache",
		"**/node_modules":                     "node_modules directories",
	}
	for path, wantReason := range wantEntries {
		gotReason, ok := exclusionConfigHasPath(cfg, path)
		if !ok {
			t.Fatalf("exclude_paths missing %q", path)
		}
		if strings.TrimSpace(gotReason) == "" {
			t.Fatalf("exclude_paths[%q] missing reason", path)
		}
		if gotReason != wantReason {
			t.Fatalf("exclude_paths[%q].reason = %q, want %q", path, gotReason, wantReason)
		}
	}
}

func excludedSection(combined string) string {
	idx := strings.Index(combined, "EXCLUDED")
	if idx < 0 {
		return ""
	}
	return combined[idx:]
}

func dotFilesSection(combined string) string {
	idxFiles := strings.Index(combined, "DOT FILES")
	idxDirs := strings.Index(combined, "DOT DIRS")
	if idxFiles < 0 || idxDirs < 0 || idxFiles >= idxDirs {
		return ""
	}
	return combined[idxFiles:idxDirs]
}

func dotDirsSection(combined string) string {
	idxDirs := strings.Index(combined, "DOT DIRS")
	idxExcluded := strings.Index(combined, "EXCLUDED")
	if idxDirs < 0 || idxExcluded < 0 || idxDirs >= idxExcluded {
		return ""
	}
	return combined[idxDirs:idxExcluded]
}

func assertDotDirsExcludes(t *testing.T, combined string, paths ...string) {
	t.Helper()
	section := dotDirsSection(combined)
	if section == "" {
		t.Fatalf("missing DOT DIRS section; got:\n%s", combined)
	}
	for _, p := range paths {
		if strings.Contains(section, p) {
			t.Fatalf("DOT DIRS unexpectedly contains %q; section:\n%s", p, section)
		}
	}
}

func dotDirsSummarySection(combined string) string {
	idxPlan := strings.Index(combined, "dry-run: machine backup plan")
	if idxPlan < 0 {
		return ""
	}
	rest := combined[idxPlan:]
	idxDirs := strings.Index(rest, "  DOT DIRS")
	idxExcluded := strings.Index(rest, "  EXCLUDED")
	if idxDirs < 0 || idxExcluded < 0 || idxDirs >= idxExcluded {
		return ""
	}
	return rest[idxDirs:idxExcluded]
}

var dotDirSummaryRowRE = regexp.MustCompile(`^\s+(\S+)\s+(\d+)\s+(\d+(?:\.\d+)?\s*(?:B|KB|MB|GB))(?:\s+LARGE SIZE)?\s*$`)

func parseDotDirSummaryRows(section string) []struct {
	Path  string
	Files int
	Size  string
	Bytes int64
} {
	var rows []struct {
		Path  string
		Files int
		Size  string
		Bytes int64
	}
	for _, line := range strings.Split(section, "\n") {
		m := dotDirSummaryRowRE.FindStringSubmatch(line)
		if len(m) < 4 {
			continue
		}
		files, _ := strconv.Atoi(m[2])
		rows = append(rows, struct {
			Path  string
			Files int
			Size  string
			Bytes int64
		}{Path: m[1], Files: files, Size: m[3], Bytes: parseHumanSizeToken(m[3])})
	}
	return rows
}

func parseHumanSizeToken(size string) int64 {
	size = strings.TrimSpace(size)
	re := regexp.MustCompile(`^(\d+(?:\.\d+)?)\s*(B|KB|MB|GB)$`)
	m := re.FindStringSubmatch(size)
	if len(m) < 3 {
		return 0
	}
	val, err := strconv.ParseFloat(m[1], 64)
	if err != nil {
		return 0
	}
	mult := int64(1)
	switch m[2] {
	case "KB":
		mult = 1024
	case "MB":
		mult = 1024 * 1024
	case "GB":
		mult = 1024 * 1024 * 1024
	}
	return int64(val * float64(mult))
}

func assertDotDirsSortedBySizeDesc(t *testing.T, combined string) {
	t.Helper()
	section := dotDirsSummarySection(combined)
	if section == "" {
		t.Fatalf("missing summary DOT DIRS section; got:\n%s", combined)
	}
	rows := parseDotDirSummaryRows(section)
	if len(rows) < 2 {
		t.Fatalf("want >=2 DOT DIRS summary rows for sort check; section:\n%s", section)
	}
	for i := 1; i < len(rows); i++ {
		if rows[i-1].Bytes < rows[i].Bytes {
			t.Fatalf("DOT DIRS not sorted by size desc: %s (%d) before %s (%d); section:\n%s",
				rows[i-1].Path, rows[i-1].Bytes, rows[i].Path, rows[i].Bytes, section)
		}
		if rows[i-1].Bytes == rows[i].Bytes && rows[i-1].Path > rows[i].Path {
			t.Fatalf("DOT DIRS tiebreak not path asc at equal size: %s before %s; section:\n%s",
				rows[i-1].Path, rows[i].Path, section)
		}
	}
}

func assertSummaryDirHasLargeSize(t *testing.T, combined, dir string) {
	t.Helper()
	section := dotDirsSummarySection(combined)
	if section == "" {
		t.Fatalf("missing summary DOT DIRS section; got:\n%s", combined)
	}
	for _, line := range strings.Split(section, "\n") {
		if !strings.Contains(line, dir) {
			continue
		}
		if strings.Contains(line, "LARGE SIZE") {
			return
		}
		t.Fatalf("DOT DIRS row for %q missing LARGE SIZE; line:\n%s", dir, line)
	}
	t.Fatalf("DOT DIRS summary missing dir %q; section:\n%s", dir, section)
}

func assertSummaryDirLacksLargeSize(t *testing.T, combined, dir string) {
	t.Helper()
	section := dotDirsSummarySection(combined)
	if section == "" {
		t.Fatalf("missing summary DOT DIRS section; got:\n%s", combined)
	}
	for _, line := range strings.Split(section, "\n") {
		if strings.Contains(line, dir) && strings.Contains(line, "LARGE SIZE") {
			t.Fatalf("DOT DIRS row for %q unexpectedly has LARGE SIZE; line:\n%s", dir, line)
		}
	}
}

func largeDirDetailSection(combined string) string {
	idx := strings.Index(combined, "LARGE DIR DETAIL:")
	if idx < 0 {
		return ""
	}
	rest := combined[idx+len("LARGE DIR DETAIL:"):]
	idxExcluded := strings.Index(rest, "  EXCLUDED")
	if idxExcluded < 0 {
		return rest
	}
	return rest[:idxExcluded]
}

var largeDirDetailLineRE = regexp.MustCompile(`^\s+>\s+(\S+)\s+(\d+(?:\.\d+)?\s*(?:B|KB|MB|GB))\s*$`)

type largeDirDetailRow struct {
	Path  string
	Size  string
	Bytes int64
}

func parseLargeDirDetailLines(section string) []largeDirDetailRow {
	var rows []largeDirDetailRow
	for _, line := range strings.Split(section, "\n") {
		m := largeDirDetailLineRE.FindStringSubmatch(line)
		if len(m) < 3 {
			continue
		}
		rows = append(rows, largeDirDetailRow{
			Path:  m[1],
			Size:  strings.TrimSpace(m[2]),
			Bytes: parseHumanSizeToken(m[2]),
		})
	}
	return rows
}

func assertLargeDirDetailFlatSorted(t *testing.T, combined string) []largeDirDetailRow {
	t.Helper()
	section := largeDirDetailSection(combined)
	if section == "" {
		t.Fatalf("missing LARGE DIR DETAIL section; got:\n%s", combined)
	}
	rows := parseLargeDirDetailLines(section)
	if len(rows) == 0 {
		t.Fatalf("LARGE DIR DETAIL has no flat detail lines; section:\n%s", section)
	}
	for i := 1; i < len(rows); i++ {
		if rows[i-1].Bytes < rows[i].Bytes {
			t.Fatalf("LARGE DIR DETAIL not sorted by size desc: %s (%d) before %s (%d); section:\n%s",
				rows[i-1].Path, rows[i-1].Bytes, rows[i].Path, rows[i].Bytes, section)
		}
		if rows[i-1].Bytes == rows[i].Bytes && rows[i-1].Path > rows[i].Path {
			t.Fatalf("LARGE DIR DETAIL tiebreak not path asc at equal size: %s before %s; section:\n%s",
				rows[i-1].Path, rows[i].Path, section)
		}
	}
	return rows
}

func assertLargeDirDetailHasPaths(t *testing.T, combined string, paths ...string) {
	t.Helper()
	rows := assertLargeDirDetailFlatSorted(t, combined)
	seen := make(map[string]largeDirDetailRow, len(rows))
	for _, row := range rows {
		seen[row.Path] = row
	}
	for _, path := range paths {
		row, ok := seen[path]
		if !ok {
			t.Fatalf("LARGE DIR DETAIL missing path %q; rows=%v\nsection:\n%s", path, rows, largeDirDetailSection(combined))
		}
		if row.Bytes < largeDirDetailThresholdBytes {
			t.Fatalf("LARGE DIR DETAIL path %q size %s (%d B) below 10 MB detail threshold",
				path, row.Size, row.Bytes)
		}
	}
}

func assertLargeDirDetailLacksPaths(t *testing.T, combined string, paths ...string) {
	t.Helper()
	section := largeDirDetailSection(combined)
	if section == "" {
		return
	}
	rows := parseLargeDirDetailLines(section)
	seen := make(map[string]struct{}, len(rows))
	for _, row := range rows {
		seen[row.Path] = struct{}{}
	}
	for _, path := range paths {
		if _, ok := seen[path]; ok {
			t.Fatalf("LARGE DIR DETAIL unexpectedly lists %q; section:\n%s", path, section)
		}
	}
}

func assertLargeDirDetailContains(t *testing.T, combined string, dir string, children ...string) {
	t.Helper()
	paths := make([]string, 0, 1+len(children))
	paths = append(paths, dir)
	for _, child := range children {
		child = strings.TrimPrefix(child, "> ")
		child = strings.TrimSpace(child)
		if strings.Contains(child, "/") {
			paths = append(paths, child)
			continue
		}
		dirBase := strings.TrimSuffix(dir, "/")
		paths = append(paths, dirBase+"/"+child)
	}
	assertLargeDirDetailHasPaths(t, combined, paths...)
}

func assertNoLargeDirDetail(t *testing.T, combined string) {
	t.Helper()
	if strings.Contains(combined, "LARGE DIR DETAIL:") {
		t.Fatalf("unexpected LARGE DIR DETAIL section; got:\n%s", combined)
	}
}

func archiveUserMembers(members []string) []string {
	out := make([]string, 0, len(members))
	for _, m := range members {
		m = strings.TrimPrefix(m, "./")
		if m == "manifest.json" || strings.HasPrefix(m, ".backup/") {
			continue
		}
		out = append(out, m)
	}
	sort.Strings(out)
	return out
}

func assertStringSetsEqual(t *testing.T, label string, want, got []string) {
	t.Helper()
	w := append([]string(nil), want...)
	g := append([]string(nil), got...)
	sort.Strings(w)
	sort.Strings(g)
	if len(w) != len(g) {
		t.Fatalf("%s count mismatch: want %d got %d\nwant=%v\ngot=%v", label, len(w), len(g), w, g)
	}
	for i := range w {
		if w[i] != g[i] {
			t.Fatalf("%s mismatch at %d: want %q got %q\nwant=%v\ngot=%v", label, i, w[i], g[i], w, g)
		}
	}
}

func assertExcludedMentions(t *testing.T, combined string, needles ...string) {
	t.Helper()
	section := excludedSection(combined)
	if section == "" {
		t.Fatalf("missing EXCLUDED section; got:\n%s", combined)
	}
	for _, n := range needles {
		if !strings.Contains(section, n) {
			t.Fatalf("EXCLUDED section missing %q; section:\n%s", n, section)
		}
	}
}

func assertDotFilesIncludes(t *testing.T, combined string, paths ...string) {
	t.Helper()
	section := dotFilesSection(combined)
	if section == "" {
		t.Fatalf("missing DOT FILES section; got:\n%s", combined)
	}
	for _, p := range paths {
		if !strings.Contains(section, p) {
			t.Fatalf("DOT FILES missing %q; section:\n%s", p, section)
		}
	}
}

func assertDotFilesExcludes(t *testing.T, combined string, paths ...string) {
	t.Helper()
	section := dotFilesSection(combined)
	if section == "" {
		t.Fatalf("missing DOT FILES section; got:\n%s", combined)
	}
	for _, p := range paths {
		if strings.Contains(section, p) {
			t.Fatalf("DOT FILES unexpectedly contains %q; section:\n%s", p, section)
		}
	}
}

func seedBackupMeta(t *testing.T, home string) {
	t.Helper()
	writeServerFile(t, home, ".backup/config.json", seededBackupMetaJSON)
}

func parseExclusionConfigJSON(t *testing.T, raw []byte) exclusionConfigJSON {
	t.Helper()
	var cfg exclusionConfigJSON
	if err := json.Unmarshal(raw, &cfg); err != nil {
		t.Fatalf("parse exclusion config: %v\n%s", err, raw)
	}
	return cfg
}

func writeServerFile(t *testing.T, home, rel, content string) {
	t.Helper()
	full := filepath.Join(home, filepath.FromSlash(rel))
	if err := os.MkdirAll(filepath.Dir(full), 0755); err != nil {
		t.Fatalf("mkdir %s: %v", filepath.Dir(full), err)
	}
	if err := os.WriteFile(full, []byte(content), 0644); err != nil {
		t.Fatalf("write %s: %v", full, err)
	}
}

func readServerFile(t *testing.T, home, rel string) string {
	t.Helper()
	full := filepath.Join(home, filepath.FromSlash(rel))
	data, err := os.ReadFile(full)
	if err != nil {
		t.Fatalf("read %s: %v", full, err)
	}
	return string(data)
}

func tarXZListMembers(t *testing.T, archivePath string) []string {
	t.Helper()
	cmd := exec.Command("tar", "-tJf", archivePath)
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("tar -tJf %s: %v\n%s", archivePath, err, out)
	}
	lines := strings.Split(strings.TrimSpace(string(out)), "\n")
	if len(lines) == 1 && lines[0] == "" {
		return nil
	}
	return lines
}

func tarXZExtractFile(t *testing.T, archivePath, member string) []byte {
	t.Helper()
	dir, err := os.MkdirTemp("", "machine-backup-extract-*")
	if err != nil {
		t.Fatalf("mkdir extract: %v", err)
	}
	t.Cleanup(func() { os.RemoveAll(dir) })
	cmd := exec.Command("tar", "-xJf", archivePath, "-C", dir, member)
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("tar extract %s from %s: %v\n%s", member, archivePath, err, out)
	}
	data, err := os.ReadFile(filepath.Join(dir, filepath.FromSlash(member)))
	if err != nil {
		t.Fatalf("read extracted %s: %v", member, err)
	}
	return data
}

func archiveHasXZMagic(t *testing.T, path string) {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read archive: %v", err)
	}
	if len(data) < 6 {
		t.Fatalf("archive too short (%d bytes)", len(data))
	}
	magic := []byte{0xFD, 0x37, 0x7A, 0x58, 0x5A, 0x00}
	if !bytes.Equal(data[:6], magic) {
		t.Fatalf("missing xz magic in %s (got % x)", path, data[:6])
	}
}

func memberListContains(members []string, want string) bool {
	want = strings.TrimPrefix(want, "./")
	for _, m := range members {
		m = strings.TrimPrefix(m, "./")
		if m == want {
			return true
		}
	}
	return false
}

func combinedHasAll(t *testing.T, combined string, needles ...string) {
	t.Helper()
	for _, n := range needles {
		if !strings.Contains(combined, n) {
			t.Fatalf("output missing %q;\nhave:\n%s", n, combined)
		}
	}
}

func assertCacheExclusionReason(t *testing.T, combined string) {
	t.Helper()
	idx := strings.Index(combined, "EXCLUDED")
	if idx < 0 {
		t.Fatalf("missing EXCLUDED section; got:\n%s", combined)
	}
	section := combined[idx:]
	lines := strings.Split(section, "\n")
	for _, line := range lines {
		if !strings.Contains(line, ".cache") {
			continue
		}
		lower := strings.ToLower(line)
		if strings.Contains(lower, "cache") || strings.Contains(lower, "temporary") {
			return
		}
	}
	t.Fatalf("EXCLUDED section missing reason for .cache; got:\n%s", section)
}

func combinedHasNone(t *testing.T, combined string, needles ...string) {
	t.Helper()
	for _, n := range needles {
		if strings.Contains(combined, n) {
			t.Fatalf("output unexpectedly contains %q;\nhave:\n%s", n, combined)
		}
	}
}

func paddedBytes(n int) []byte {
	b := make([]byte, n)
	for i := range b {
		b[i] = 'x'
	}
	return b
}

// seedExcludedSizesFixtures overwrites cache/log fixtures with known byte sizes so
// EXCLUDED per-rule FILES and SIZE columns are deterministic (1024+512 B under .cache,
// 512 B under **/*.log).
func seedExcludedSizesFixtures(t *testing.T, home string) {
	t.Helper()
	writeServerBinaryFile(t, home, ".cache/junk", paddedBytes(1024))
	writeServerBinaryFile(t, home, ".cache/nested/deep", paddedBytes(512))
	writeServerBinaryFile(t, home, ".ai-critic/service.log", paddedBytes(512))
}

var (
	backupSizeToken       = regexp.MustCompile(`\d+(\.\d+)?\s*(B|KB|MB)\b`)
	excludedStatsHeaderRE = regexp.MustCompile(`EXCLUDED \(\d+ paths, \d+ files,`)
	excludedRuleRowRE     = regexp.MustCompile(`^\s+(\S.*?)\s+(\d+)\s+(\d+(?:\.\d+)?\s*(?:B|KB|MB))\s+`)
)

func assertExcludedStatsHeader(t *testing.T, combined string) {
	t.Helper()
	section := excludedSection(combined)
	if section == "" {
		t.Fatalf("missing EXCLUDED section; got:\n%s", combined)
	}
	if !excludedStatsHeaderRE.MatchString(section) {
		t.Fatalf("EXCLUDED section missing paths/files/size header; section:\n%s", section)
	}
	if !backupSizeToken.MatchString(section) {
		t.Fatalf("EXCLUDED header missing size token; section:\n%s", section)
	}
}

func assertExcludedTableHeaders(t *testing.T, combined string) {
	t.Helper()
	section := excludedSection(combined)
	if !strings.Contains(section, "RULE") || !strings.Contains(section, "FILES") {
		t.Fatalf("EXCLUDED section missing RULE/FILES column headers; section:\n%s", section)
	}
}

func excludedRuleLineIndex(section, rule string) int {
	lines := strings.Split(section, "\n")
	for i, line := range lines {
		if strings.Contains(line, rule) && excludedRuleRowRE.MatchString(line) {
			return i
		}
	}
	return -1
}

func assertExcludedRuleBefore(t *testing.T, combined, firstRule, secondRule string) {
	t.Helper()
	section := excludedSection(combined)
	i := excludedRuleLineIndex(section, firstRule)
	j := excludedRuleLineIndex(section, secondRule)
	if i < 0 {
		t.Fatalf("EXCLUDED section missing rule row %q; section:\n%s", firstRule, section)
	}
	if j < 0 {
		t.Fatalf("EXCLUDED section missing rule row %q; section:\n%s", secondRule, section)
	}
	if i >= j {
		t.Fatalf("expected %q row before %q row (lines %d vs %d); section:\n%s", firstRule, secondRule, i, j, section)
	}
}

func parseExcludedRuleRow(line string) (rule string, files int, size string, ok bool) {
	m := excludedRuleRowRE.FindStringSubmatch(line)
	if len(m) < 4 {
		return "", 0, "", false
	}
	files, err := strconv.Atoi(m[2])
	if err != nil {
		return "", 0, "", false
	}
	return strings.TrimSpace(m[1]), files, strings.TrimSpace(m[3]), true
}

func assertExcludedRuleFilesAtLeast(t *testing.T, combined, rule string, minFiles int) {
	t.Helper()
	section := excludedSection(combined)
	for _, line := range strings.Split(section, "\n") {
		if !strings.Contains(line, rule) {
			continue
		}
		gotRule, files, _, ok := parseExcludedRuleRow(line)
		if !ok || !strings.Contains(gotRule, rule) {
			continue
		}
		if files >= minFiles {
			return
		}
		t.Fatalf("EXCLUDED rule %q files=%d, want >= %d; line:\n%s", rule, files, minFiles, line)
	}
	t.Fatalf("EXCLUDED section missing rule row %q; section:\n%s", rule, section)
}

func assertExcludedRuleHasSizeKB(t *testing.T, combined, rule string) {
	t.Helper()
	section := excludedSection(combined)
	for _, line := range strings.Split(section, "\n") {
		if !strings.Contains(line, rule) {
			continue
		}
		gotRule, _, size, ok := parseExcludedRuleRow(line)
		if !ok || !strings.Contains(gotRule, rule) {
			continue
		}
		if strings.Contains(size, "KB") || strings.Contains(size, "MB") {
			return
		}
		t.Fatalf("EXCLUDED rule %q size=%q, want >= 1 KB; line:\n%s", rule, size, line)
	}
	t.Fatalf("EXCLUDED section missing rule row %q; section:\n%s", rule, section)
}

type backupPlanFetchBody struct {
	DryRun                 bool     `json:"dry_run"`
	Exclude                []string `json:"exclude"`
	Include                []string `json:"include"`
	LargeDirThresholdBytes int64    `json:"large_dir_threshold_bytes,omitempty"`
}

func fetchBackupPlanIncluded(t *testing.T, serverURL, token string, exclude, include []string, largeDirThreshold int64) []string {
	t.Helper()
	body := backupPlanFetchBody{DryRun: true, Exclude: exclude, Include: include}
	if largeDirThreshold > 0 {
		body.LargeDirThresholdBytes = largeDirThreshold
	}
	raw, err := json.Marshal(body)
	if err != nil {
		t.Fatalf("marshal backup plan body: %v", err)
	}
	backupURL := strings.TrimRight(strings.TrimSpace(serverURL), "/") + "/api/remote-agent/machine/backup"
	req, err := http.NewRequest(http.MethodPost, backupURL, bytes.NewReader(raw))
	if err != nil {
		t.Fatalf("build backup plan request: %v", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+token)
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("fetch backup plan: %v", err)
	}
	defer resp.Body.Close()
	data, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatalf("read backup plan response: %v", err)
	}
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("backup plan status %d: %s", resp.StatusCode, strings.TrimSpace(string(data)))
	}
	var plan struct {
		Included []string `json:"included"`
	}
	if err := json.Unmarshal(data, &plan); err != nil {
		t.Fatalf("decode backup plan: %v\n%s", err, data)
	}
	return plan.Included
}

func subcommandFlagsFromRequest(req *Request) []string {
	var flags []string
	for _, ex := range req.ExcludePaths {
		flags = append(flags, "--exclude", ex)
	}
	for _, inc := range req.IncludePaths {
		flags = append(flags, "--include", inc)
	}
	if req.ShowConfig {
		flags = append(flags, "--show-config")
	}
	if req.ShowMeta {
		flags = append(flags, "--show-meta")
	}
	if req.SetConfig {
		flags = append(flags, "--set-config")
	}
	if req.SetConfigLargeDirThreshold != "" {
		flags = append(flags, "--large-dir-threshold", req.SetConfigLargeDirThreshold)
	}
	if req.SkipGitDirsScan {
		flags = append(flags, "--skip-git-dirs-scan")
	}
	if req.GitDirsScanMaxDepth > 0 {
		flags = append(flags, "--git-dirs-scan-max-depth", strconv.Itoa(req.GitDirsScanMaxDepth))
	}
	return flags
}

func argvWithoutDryRun(argv []string) []string {
	out := make([]string, 0, len(argv))
	for i := 0; i < len(argv); i++ {
		if argv[i] == "--dry-run" {
			continue
		}
		out = append(out, argv[i])
	}
	return out
}

func runDryRunThenArchive(t *testing.T, req *Request, resp *Response, agentBin string, agentEnv []string, serverURL string) (*Response, error) {
	dryArgv := make([]string, 0, len(req.Args)+16)
	dryArgv = append(dryArgv, "--server", serverURL, "--token", req.Token)
	dryArgv = append(dryArgv, req.Args...)
	dryArgv = insertSubcommandFlags(dryArgv, subcommandFlagsFromRequest(req)...)

	t.Logf("dry-run argv: %v", dryArgv)
	exitCode, stdout, stderr, runErr := runAgent(agentBin, dryArgv, agentEnv)
	if runErr != nil {
		return nil, runErr
	}
	resp.ExitCode = exitCode
	resp.DryRunCombined = strings.TrimSpace(stdout + "\n" + stderr)
	resp.Combined = resp.DryRunCombined
	if exitCode != 0 {
		return resp, nil
	}

	resp.DryRunIncluded = fetchBackupPlanIncluded(t, serverURL, req.Token, req.ExcludePaths, req.IncludePaths, 0)

	if req.OutputPath == "" {
		return nil, fmt.Errorf("DryRunThenArchive requires OutputPath")
	}
	out := req.OutputPath
	if !filepath.IsAbs(out) {
		out = filepath.Join(resp.AgentHome, out)
	}
	absOut, err := filepath.Abs(out)
	if err != nil {
		return nil, err
	}
	req.OutputPath = absOut

	backupArgv := argvWithoutDryRun(dryArgv)
	backupArgv = insertSubcommandFlags(backupArgv, "--output", absOut)
	t.Logf("backup argv: %v", backupArgv)
	exitCode, stdout, stderr, runErr = runAgent(agentBin, backupArgv, agentEnv)
	if runErr != nil {
		return nil, runErr
	}
	resp.ExitCode = exitCode
	resp.Stdout = stdout
	resp.Stderr = stderr
	resp.Combined = strings.TrimSpace(stdout + "\n" + stderr)
	resp.BackupPath = absOut
	return resp, nil
}
```