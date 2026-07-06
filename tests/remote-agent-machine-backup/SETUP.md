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

Implements REQUIREMENT-DESIGN-remote-agent-machine-backup.md. Tests are expected to
fail until `machine backup` and `machine restore` are implemented.

```go
import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
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
	return req.SeedDocker || req.SeedBackupMeta
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
	Version      string `json:"version"`
	ExcludePaths []struct {
		Path   string `json:"path"`
		Reason string `json:"reason"`
	} `json:"exclude_paths"`
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
	return nil
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
```