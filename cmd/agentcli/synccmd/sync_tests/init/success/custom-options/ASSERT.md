## Expected

1. `RunErr` empty.
2. Pair `custom`: prefer `older`, localHostname `mac-x`, remoteUnison `/opt/unison`.
3. Profile exists and `servercmd` mentions `/opt/unison` (or equals).

## Side Effects

- Store + profile for `custom`.

## Errors

- None.

## Exit Code

- Nil error.

```go
import (
	"strings"
	"testing"

	"github.com/xhd2015/doctest/session"
)

func Assert(t *testing.T, d *session.Doctest, req *Request, resp *Response, err error) {
	t.Helper()
	_ = d
	_ = req
	if err != nil {
		t.Fatalf("harness Run returned unexpected error: %v", err)
	}
	if resp.RunErr != "" {
		t.Fatalf("RunCLI error: %s", resp.RunErr)
	}
	p := pairByName(resp, "custom")
	if p == nil {
		t.Fatal("pair custom missing")
	}
	if p.Prefer != "older" {
		t.Fatalf("prefer: got %q want older", p.Prefer)
	}
	if p.LocalHostname != "mac-x" {
		t.Fatalf("localHostname: got %q want mac-x", p.LocalHostname)
	}
	if p.RemoteUnison != "/opt/unison" {
		t.Fatalf("remoteUnison: got %q want /opt/unison", p.RemoteUnison)
	}
	if !resp.ProfileExists {
		t.Fatal("profile missing")
	}
	if !strings.Contains(resp.ProfileContent, "/opt/unison") {
		t.Fatalf("profile missing servercmd path; content:\\n%s", resp.ProfileContent)
	}
}
```
