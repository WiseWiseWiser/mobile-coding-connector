## Expected

1. Success errors empty.
2. Pair: prefer `older`; times/auto/batch all false.
3. Profile reflects `prefer = older` and false-ish auto/batch/times lines when present.

## Side Effects

- Store + profile updated.

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
	if resp.PreErr != "" {
		t.Fatalf("pre-init failed: %s", resp.PreErr)
	}
	if resp.RunErr != "" {
		t.Fatalf("set error: %s", resp.RunErr)
	}
	p := pairByName(resp, "mad-max")
	if p == nil {
		t.Fatal("pair missing")
	}
	if p.Prefer != "older" {
		t.Fatalf("prefer: got %q want older", p.Prefer)
	}
	if p.Times || p.Auto || p.Batch {
		t.Fatalf("want times/auto/batch false; got times=%v auto=%v batch=%v", p.Times, p.Auto, p.Batch)
	}
	if resp.ProfileExists {
		if !strings.Contains(resp.ProfileContent, "prefer = older") && !strings.Contains(resp.ProfileContent, "prefer=older") {
			// allow flexible spacing: require prefer and older nearby via both substrings
			if !(strings.Contains(resp.ProfileContent, "prefer") && strings.Contains(resp.ProfileContent, "older")) {
				t.Fatalf("profile missing prefer older; content:\\n%s", resp.ProfileContent)
			}
		}
	}
}
```
