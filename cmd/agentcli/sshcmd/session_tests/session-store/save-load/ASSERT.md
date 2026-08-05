## Expected

1. `SaveErr` and `LoadErr` are empty.
2. `Loaded` is non-nil.
3. Loaded fields match SessionToSave:
   - LocalPort, User, ConfigDir, ServePID, ProfileID, Alive.

## Side Effects

- Session file created under `{Root}/ssh-sessions/{profileID}.json`.

## Errors

None expected.

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
		t.Fatal("Loaded session is nil; want non-nil after Save")
	}
	want := req.SessionToSave
	got := resp.Loaded
	if got.LocalPort != want.LocalPort {
		t.Fatalf("LocalPort: got %d want %d", got.LocalPort, want.LocalPort)
	}
	if got.User != want.User {
		t.Fatalf("User: got %q want %q", got.User, want.User)
	}
	if got.ConfigDir != want.ConfigDir {
		t.Fatalf("ConfigDir: got %q want %q", got.ConfigDir, want.ConfigDir)
	}
	if got.ServePID != want.ServePID {
		t.Fatalf("ServePID: got %d want %d", got.ServePID, want.ServePID)
	}
	if got.ProfileID != want.ProfileID {
		t.Fatalf("ProfileID: got %q want %q", got.ProfileID, want.ProfileID)
	}
	if got.Alive != want.Alive {
		t.Fatalf("Alive: got %v want %v", got.Alive, want.Alive)
	}
}
```
