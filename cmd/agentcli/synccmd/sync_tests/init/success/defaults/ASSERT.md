## Expected

1. `RunErr` empty.
2. `pairs.json` exists; Config has one pair `mad-max`.
3. Pair fields:
   - backend `unison`
   - local/remote match request paths
   - prefer `newer`
   - localHostname `remote-agent-mad-max`
   - remoteUnison `/usr/local/bin/unison`
   - times/auto/batch true
   - ignore includes default five entries
4. Profile exists at `remote-agent-mad-max.prf`.

## Side Effects

- `{StoreDir}/pairs.json` written.
- `{UnisonDir}/remote-agent-mad-max.prf` written.

## Errors

- None.

## Exit Code

- Nil error.

```go
import (
	"path/filepath"
	"testing"

	"github.com/xhd2015/ai-critic/cmd/agentcli/synccmd"
	"github.com/xhd2015/doctest/session"
)

func Assert(t *testing.T, d *session.Doctest, req *Request, resp *Response, err error) {
	t.Helper()
	_ = d
	if err != nil {
		t.Fatalf("harness Run returned unexpected error: %v", err)
	}
	if resp.RunErr != "" {
		t.Fatalf("RunCLI error: %s", resp.RunErr)
	}
	if !resp.PairsJSONExists {
		t.Fatal("pairs.json missing after init")
	}
	p := pairByName(resp, "mad-max")
	if p == nil {
		t.Fatalf("pair mad-max missing; config=%+v json=%s", resp.Config, resp.PairsJSON)
	}
	if p.Backend != "unison" {
		t.Fatalf("backend: got %q want unison", p.Backend)
	}
	if p.Local != req.LocalPath {
		t.Fatalf("local: got %q want %q", p.Local, req.LocalPath)
	}
	if p.Remote != req.RemotePath {
		t.Fatalf("remote: got %q want %q", p.Remote, req.RemotePath)
	}
	if p.Prefer != "newer" {
		t.Fatalf("prefer: got %q want newer", p.Prefer)
	}
	if p.LocalHostname != "remote-agent-mad-max" {
		t.Fatalf("localHostname: got %q want remote-agent-mad-max", p.LocalHostname)
	}
	if p.RemoteUnison != "/usr/local/bin/unison" {
		t.Fatalf("remoteUnison: got %q", p.RemoteUnison)
	}
	if !p.Times || !p.Auto || !p.Batch {
		t.Fatalf("times/auto/batch want all true; got times=%v auto=%v batch=%v", p.Times, p.Auto, p.Batch)
	}
	for _, ign := range defaultIgnoreList() {
		found := false
		for _, g := range p.Ignore {
			if g == ign {
				found = true
				break
			}
		}
		if !found {
			t.Fatalf("ignore missing %q; got %v", ign, p.Ignore)
		}
	}
	if !resp.ProfileExists {
		t.Fatalf("profile missing at %s", resp.ProfilePath)
	}
	wantName := synccmd.ProfileFileName("mad-max")
	if filepath.Base(resp.ProfilePath) != wantName {
		t.Fatalf("profile basename: got %q want %q", filepath.Base(resp.ProfilePath), wantName)
	}
}
```
