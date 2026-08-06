## Expected

1. `PreErr`/`RunErr` empty.
2. Pair local/remote equal the v2 paths from Args.
3. Profile content contains local path as a `root =` line and remote as
   `ssh://remote-agent//` + remote path (or contains both path strings and `ssh://remote-agent`).

## Side Effects

- pairs.json updated; profile rewritten.

## Errors

- None.

## Exit Code

- Nil error.

```go
import (
	"path/filepath"
	"strings"
	"testing"

	"github.com/xhd2015/doctest/session"
)

func Assert(t *testing.T, d *session.Doctest, req *Request, resp *Response, err error) {
	t.Helper()
	if err != nil {
		t.Fatalf("harness Run returned unexpected error: %v", err)
	}
	if resp.PreErr != "" {
		t.Fatalf("pre-init failed: %s", resp.PreErr)
	}
	if resp.RunErr != "" {
		t.Fatalf("set error: %s", resp.RunErr)
	}
	p := pairByName(resp, "mad-max")
	if p == nil {
		t.Fatal("pair missing after set")
	}
	wantLocal := filepath.Join(d.DOCTEST_CASE, "workspace-local-v2")
	wantRemote := filepath.Join(d.DOCTEST_CASE, "workspace-remote-v2")
	if p.Local != wantLocal {
		t.Fatalf("local: got %q want %q", p.Local, wantLocal)
	}
	if p.Remote != wantRemote {
		t.Fatalf("remote: got %q want %q", p.Remote, wantRemote)
	}
	if !resp.ProfileExists {
		t.Fatal("profile missing after set")
	}
	c := resp.ProfileContent
	if !strings.Contains(c, wantLocal) {
		t.Fatalf("profile missing local root path; content:\\n%s", c)
	}
	if !strings.Contains(c, "ssh://remote-agent/") {
		t.Fatalf("profile missing ssh://remote-agent/ root; content:\\n%s", c)
	}
	if !strings.Contains(c, wantRemote) {
		t.Fatalf("profile missing remote path; content:\\n%s", c)
	}
}
```
