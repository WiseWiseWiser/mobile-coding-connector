# Scenario

**Feature**: client-config writes JSON to `--output` file

```
# --output FILE: config on disk, stdout quiet
BuildSingBoxTunConfig -> --output FILE
```

## Steps

1. Set `OutputFile` to a temp path so the test does not dirty the worktree.

```go
import (
	"path/filepath"
	"testing"
)

func Setup(t *testing.T, req *Request) error {
	req.OutputFile = filepath.Join(t.TempDir(), "singbox-client-config.json")
	return nil
}
```
