# Scenario

**Feature**: `--include-file` path must be part of dirty pull set

```
# small dirty change; include_files lists path not in dirty set
remote-agent project pull-local --include-file not-in-pull.bin -> exit 1
```

## Preconditions

Remote dirty with modified README only; include names absent file.

## Steps

1. `pairSameOriginRepos`; modify README.
2. Args: `--include-file not-in-pull.bin`.

## Context

REQUIREMENT leaf `include-file-not-dirty`.

```go
import (
	"os"
	"path/filepath"
	"testing"
)

func Setup(t *testing.T, req *Request) error {
	pair := pairSameOriginRepos(t)
	readme := filepath.Join(pair.RemoteDir, "README.md")
	if err := os.WriteFile(readme, []byte("small dirty\n"), 0644); err != nil {
		t.Fatalf("modify readme: %v", err)
	}
	seedSizeLimitPullProject(t, req, "pull-size-badinc-001", "pull-size-badinc", pair)
	req.Args = []string{
		"project", "pull-local", "pull-size-badinc-001",
		"--include-file", "not-in-pull.bin",
	}
	return nil
}
```