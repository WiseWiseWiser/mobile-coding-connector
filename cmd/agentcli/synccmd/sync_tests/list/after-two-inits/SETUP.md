# Scenario

**Feature**: list shows two pair names after two inits

```
PreArgs: init alpha, init beta
operator -> list -> stdout contains alpha and beta
```

## Preconditions

- Two pairs: `alpha` and `beta` with distinct absolute local/remote dirs under the case dir.

## Steps

1. Create secondary workspace dirs for beta.
2. PreArgs: init alpha then init beta.
3. Args: `unison list`.
4. Assert both names on stdout and two store entries.

## Context

- Non-empty list coverage.

```go
import (
	"os"
	"path/filepath"
	"testing"

	"github.com/xhd2015/doctest/session"
)

func Setup(t *testing.T, d *session.Doctest, req *Request) error {
	t.Helper()
	localB := filepath.Join(d.DOCTEST_CASE, "workspace-local-b")
	remoteB := filepath.Join(d.DOCTEST_CASE, "workspace-remote-b")
	for _, dir := range []string{localB, remoteB} {
		if err := os.MkdirAll(dir, 0o755); err != nil {
			return err
		}
	}
	req.PreArgs = [][]string{
		{"unison", "init", "alpha", req.LocalPath, req.RemotePath},
		{"unison", "init", "beta", localB, remoteB},
	}
	req.Args = []string{"unison", "list"}
	return nil
}
```
