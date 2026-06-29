---
label: slow
explanation: writes 65 MiB of fixture data and extracts tarball to worktree
---

## Expected

1. Exit code 0.
2. Worktree contains at least one `bulk/chunk-*.bin` of size 1 MiB.
3. Remote porcelain empty after truncate.

## Side Effects

Large worktree; remote cleaned.

## Errors

- Exit non-zero or missing bulk artifacts.

## Exit Code

0.

```go
import (
	"os"
	"path/filepath"
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
	base := worktreeBaseDir(resp.AgentHome)
	wtPath := findNewestWorktreeDirSize(t, base)
	chunk0 := filepath.Join(wtPath, "bulk", "chunk-000.bin")
	info, err := os.Stat(chunk0)
	if err != nil {
		t.Fatalf("chunk-000.bin in worktree: %v", err)
	}
	if info.Size() < perFileCapBytes {
		t.Fatalf("chunk size %d want >= %d", info.Size(), perFileCapBytes)
	}
	if porcelain := gitPorcelain(t, resp.ProjectDir); strings.TrimSpace(porcelain) != "" {
		t.Fatalf("remote still dirty:\n%s", porcelain)
	}
}

func findNewestWorktreeDirSize(t *testing.T, base string) string {
	t.Helper()
	var candidates []string
	_ = filepath.WalkDir(base, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() && strings.Contains(filepath.Base(path), "main-") {
			if _, err := os.Stat(filepath.Join(path, ".git")); err == nil {
				candidates = append(candidates, path)
			}
		}
		return nil
	})
	if len(candidates) == 0 {
		t.Fatalf("no main-* worktree under %s", base)
	}
	return candidates[len(candidates)-1]
}
```