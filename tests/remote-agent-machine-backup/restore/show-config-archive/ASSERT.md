## Expected Output

Stdout is effective exclusion config from the prereq backup archive.

## Expected

1. Exit code 0.
2. Stdout parses as exclusion config with `version` `1.0`.
3. `exclude_paths` matches archive `.backup/config.json` (includes `.cache`, `.npm`).
4. No restore apply side effects on `serverHome` dot fixtures.

## Side Effects

None (read-only archive inspection).

## Errors

- Config missing built-in exclusions present at backup time.

## Exit Code

0.

```go
import (
	"strings"
	"testing"
)

func Assert(t *testing.T, req *Request, resp *Response, err error) {
	if err != nil {
		t.Fatalf("Run error: %v", err)
	}
	if resp.ExitCode != 0 {
		t.Fatalf("exit %d; combined:\n%s", resp.ExitCode, resp.Combined)
	}
	if resp.BackupPath == "" {
		t.Fatal("prereq BackupPath empty")
	}

	archiveCfgRaw := tarXZExtractFile(t, resp.BackupPath, ".backup/config.json")
	archiveCfg := parseExclusionConfigJSON(t, archiveCfgRaw)

	stdoutCfg := parseExclusionConfigJSON(t, []byte(strings.TrimSpace(resp.Stdout)))
	if stdoutCfg.Version != archiveCfg.Version {
		t.Fatalf("stdout version %q != archive %q", stdoutCfg.Version, archiveCfg.Version)
	}
	if len(stdoutCfg.ExcludePaths) != len(archiveCfg.ExcludePaths) {
		t.Fatalf("exclude_paths count %d != archive %d", len(stdoutCfg.ExcludePaths), len(archiveCfg.ExcludePaths))
	}

	paths := make(map[string]string, len(archiveCfg.ExcludePaths))
	for _, e := range archiveCfg.ExcludePaths {
		paths[e.Path] = e.Reason
	}
	for _, e := range stdoutCfg.ExcludePaths {
		wantReason, ok := paths[e.Path]
		if !ok {
			t.Fatalf("stdout has unexpected path %q", e.Path)
		}
		if e.Reason != wantReason {
			t.Fatalf("reason for %q = %q, want %q", e.Path, e.Reason, wantReason)
		}
	}

	foundCache, foundNPM := false, false
	for _, e := range stdoutCfg.ExcludePaths {
		switch e.Path {
		case ".cache":
			foundCache = true
		case ".npm":
			foundNPM = true
		}
	}
	if !foundCache || !foundNPM {
		t.Fatalf("stdout config missing expected built-in paths: %+v", stdoutCfg.ExcludePaths)
	}

	// Server home unchanged from seed (no restore apply).
	if got := readServerFile(t, resp.ServerHome, ".bashrc"); got != "export FAKE=1\n" {
		t.Fatalf(".bashrc mutated during show-config: %q", got)
	}
}
```