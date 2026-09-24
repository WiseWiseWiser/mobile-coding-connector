# Scenario

**Feature**: product binaries round-trip an edit (L3 smoke)

```
# real ai-critic-server + remote-agent edit with a fake editor -> remote updated
ai-critic-server (HOME=serverHome) + remote-agent -> edit notes.md -> Saved
```

## Preconditions

Session cache holds freshly built `ai-critic-server` and `remote-agent` binaries
(shared, flock-guarded). Remote `notes.md` contains `old\n`.

## Steps

1. `Run` builds/starts the product server with `HOME=serverHome` and a
   credentials file, and writes the `remote-agent` config with server + token.
2. `remote-agent edit notes.md` runs with the generated editor script, `--work-dir`,
   and `HOME=agentHome`.
3. Assert exit 0, `Saved`, and updated remote bytes.

## Context

Sparse L3 smoke — e2e/roundtrip. Keeps the product binary path (config file
resolution, HTTP auth, multipart write) covered by at least one leaf.

```go
import (
	"testing"

	"github.com/xhd2015/doctest/session"
)

func Setup(t *testing.T, _ *session.Doctest, req *Request) error {
	req.UseCLI = true
	setEditArgs(req, standardRemoteFile, standardRemoteInitial)
	req.EditorWrite = "e2e\n"
	return nil
}
```
