## Expected

1. `SaveErr` and `LoadErr` are empty.
2. `Loaded` is non-nil (session file still readable).
3. `Loaded.Alive` is **false** — dead ServePID must not count as active tunnel.
4. Other identity fields remain (ProfileID, LocalPort at least).

## Side Effects

- None beyond the session file written for the test.

## Errors

None expected from Save/Load I/O.

```go
import (
	"testing"

	"github.com/xhd2015/doctest/session"
)

func Assert(t *testing.T, d *session.Doctest, req *Request, resp *Response, err error) {
	t.Helper()
	_ = d
	if err != nil {
		t.Fatalf("harness Run returned unexpected error: %v", err)
	}
	if resp.SaveErr != "" {
		t.Fatalf("Save error: %s", resp.SaveErr)
	}
	if resp.LoadErr != "" {
		t.Fatalf("Load error: %s", resp.LoadErr)
	}
	if resp.Loaded == nil {
		t.Fatal("Loaded is nil; want session with Alive=false for dead ServePID")
	}
	if resp.Loaded.Alive {
		t.Fatalf("Alive must be false when ServePID %d is dead; got Alive=true", req.SessionToSave.ServePID)
	}
	if resp.Loaded.ProfileID != req.ProfileID {
		t.Fatalf("ProfileID: got %q want %q", resp.Loaded.ProfileID, req.ProfileID)
	}
	if resp.Loaded.LocalPort != req.SessionToSave.LocalPort {
		t.Fatalf("LocalPort: got %d want %d", resp.Loaded.LocalPort, req.SessionToSave.LocalPort)
	}
}
```
