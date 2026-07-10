## Expected

1. HTTP status `200`.
2. Exactly one project.
3. `clean` is `true`.
4. `error` is empty/absent.
5. `name` equals `filepath.Base` of the project path.
6. `worktrees` is empty (length 0).
7. `branch` is non-empty (e.g. `main`).

## Errors

- Dirty or error flags on a clean main.
- Linked worktrees listed when none exist.

```go
import (
	"path/filepath"
	"testing"
)

func Assert(t *testing.T, req *Request, resp *Response, err error) {
	if err != nil {
		t.Fatal(err)
	}
	if resp.StatusCode != 200 {
		t.Fatalf("status = %d, want 200; body=%s", resp.StatusCode, resp.Body)
	}
	if len(resp.Projects) != 1 {
		t.Fatalf("projects len = %d, want 1; body=%s", len(resp.Projects), resp.Body)
	}
	p := resp.Projects[0]
	if clean, _ := p["clean"].(bool); !clean {
		t.Fatalf("clean = %v, want true; project=%v", p["clean"], p)
	}
	if errStr, _ := p["error"].(string); errStr != "" {
		t.Fatalf("error = %q, want empty; project=%v", errStr, p)
	}
	path, _ := p["path"].(string)
	name, _ := p["name"].(string)
	if name == "" || name != filepath.Base(path) {
		t.Fatalf("name = %q, want Base(%q)=%q", name, path, filepath.Base(path))
	}
	branch, _ := p["branch"].(string)
	if branch == "" {
		t.Fatalf("branch empty; project=%v", p)
	}
	wts := worktreesOf(p)
	if len(wts) != 0 {
		t.Fatalf("worktrees len = %d, want 0; worktrees=%v", len(wts), wts)
	}
}

func worktreesOf(p map[string]any) []map[string]any {
	raw, ok := p["worktrees"]
	if !ok || raw == nil {
		return nil
	}
	arr, ok := raw.([]any)
	if !ok {
		return nil
	}
	var out []map[string]any
	for _, item := range arr {
		if m, ok := item.(map[string]any); ok {
			out = append(out, m)
		}
	}
	return out
}
```
