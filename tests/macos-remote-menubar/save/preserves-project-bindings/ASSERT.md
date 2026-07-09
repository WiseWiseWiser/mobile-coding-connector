## Expected

1. `SavedOK` is true.
2. Reloaded `ProjectBindingsJSON` still contains the original binding
   (`remote_dir` `/home/u/proj`, `local_path` `/Users/u/proj`, server
   `https://example.com`).

## Side Effects

Writes a temp `remote-agent-config.json` only (cleaned up after test).

## Errors

- Dropping `project_bindings` on save (common when re-marshaling a partial struct).

```go
import (
	"encoding/json"
	"strings"
	"testing"
)

func Assert(t *testing.T, req *Request, resp *Response, err error) {
	if err != nil {
		t.Fatal(err)
	}
	if !resp.SavedOK {
		t.Fatal("expected SavedOK")
	}
	var bindings []map[string]string
	if err := json.Unmarshal([]byte(resp.ProjectBindingsJSON), &bindings); err != nil {
		t.Fatalf("ProjectBindingsJSON not valid JSON: %v (%q)", err, resp.ProjectBindingsJSON)
	}
	if len(bindings) != 1 {
		t.Fatalf("bindings len = %d, want 1; raw=%s", len(bindings), resp.ProjectBindingsJSON)
	}
	b := bindings[0]
	if b["server"] != "https://example.com" {
		t.Fatalf("binding.server = %q", b["server"])
	}
	if b["remote_dir"] != "/home/u/proj" {
		t.Fatalf("binding.remote_dir = %q", b["remote_dir"])
	}
	if b["local_path"] != "/Users/u/proj" {
		t.Fatalf("binding.local_path = %q", b["local_path"])
	}
	// Sanity: token update should not appear inside bindings JSON.
	if strings.Contains(resp.ProjectBindingsJSON, "new-secret") {
		t.Fatal("token leaked into project_bindings JSON")
	}
}
```
