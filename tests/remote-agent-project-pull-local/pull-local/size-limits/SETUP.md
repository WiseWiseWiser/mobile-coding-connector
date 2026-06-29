# Scenario

**Feature**: server pull-local size guards and CLI `--include-file` / `--max-size`

```
# dirty remote + byte accounting on server -> 400 or tar.gz -> CLI exit / worktree
remote-agent project pull-local [flags] -> POST pull-local (dry_run then package) -> local apply
```

## Preconditions

Binding and same-origin local repo; remote worktree dirty with controlled file sizes.

## Steps

1. Leaf seeds oversized untracked files or many sub-1MB files via `writeBigFile`.
2. Leaf sets `project pull-local` argv including optional `--include-file` / `--max-size`.
3. `Assert` checks exit code, user-facing hints, and worktree presence for success leaves.

## Context

REQUIREMENT server-side pull-local API size limits. Per-file cap 1MB (1048576);
default total cap 64MB unless overridden.

```go
import (
	"fmt"
	"os"
	"path/filepath"
	"testing"
)

func Setup(t *testing.T, req *Request) error {
	if len(req.Args) >= 2 && req.Args[1] != "pull-local" {
		t.Fatalf("size-limits group: unexpected argv %v", req.Args)
	}
	return nil
}

// writeBigFile creates or overwrites repo-relative path with at least sizeBytes of payload.
func writeBigFile(t *testing.T, dir, rel string, sizeBytes int) {
	t.Helper()
	if sizeBytes < 0 {
		t.Fatalf("writeBigFile: negative size %d", sizeBytes)
	}
	full := filepath.Join(dir, filepath.FromSlash(rel))
	if err := os.MkdirAll(filepath.Dir(full), 0755); err != nil {
		t.Fatalf("mkdir for %s: %v", rel, err)
	}
	payload := make([]byte, sizeBytes)
	for i := range payload {
		payload[i] = byte('x' + (i % 26))
	}
	if err := os.WriteFile(full, payload, 0644); err != nil {
		t.Fatalf("write %s (%d bytes): %v", rel, sizeBytes, err)
	}
}

// seedSizeLimitPullProject sets binding + project id for size-limit leaves.
func seedSizeLimitPullProject(t *testing.T, req *Request, id, name string, pair RepoPair) {
	t.Helper()
	registerPullProject(t, req, id, name, pair.RemoteDir)
	seedBindingForServer(t, req, pair.RemoteDir, pair.LocalDir)
}

const (
	perFileCapBytes      = 1 << 20 // 1 MiB
	defaultMaxTotalBytes = 64 << 20
	bigUntrackedRel      = "big.bin"
	bulkFileCountOver64  = 65 // 65 MiB untracked > default 64 MiB cap
)

// dirtyWithBigUntracked marks top-level dirty and adds a 2 MiB untracked big.bin.
func dirtyWithBigUntracked(t *testing.T, remoteDir string) {
	t.Helper()
	readme := filepath.Join(remoteDir, "README.md")
	if err := os.WriteFile(readme, []byte("dirty for size test\n"), 0644); err != nil {
		t.Fatalf("modify readme: %v", err)
	}
	writeBigFile(t, remoteDir, bigUntrackedRel, 2*perFileCapBytes)
}

// dirtyWithManyOneMiBFiles adds count untracked files each 1 MiB (paths bulk/chunk-NNN.bin).
func dirtyWithManyOneMiBFiles(t *testing.T, remoteDir string, count int) {
	t.Helper()
	readme := filepath.Join(remoteDir, "README.md")
	if err := os.WriteFile(readme, []byte("dirty bulk\n"), 0644); err != nil {
		t.Fatalf("modify readme: %v", err)
	}
	for i := 0; i < count; i++ {
		rel := filepath.Join("bulk", fmt.Sprintf("chunk-%03d.bin", i))
		writeBigFile(t, remoteDir, rel, perFileCapBytes)
	}
}
```