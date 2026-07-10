## Expected

1. HTTP status `200`.
2. One project; main `clean` is `true` (main checkout not dirtied).
3. At least one linked worktree entry.
4. Linked entry has non-empty `path` and `name` (basename).
5. Linked entry `is_main` is `false`.
6. At least one linked worktree has `clean=false` (the dirty one).

## Errors

- Missing worktrees when linked exists.
- Dirty main when only linked was dirtied.
- Linked listed as `is_main=true`.

```go
import "testing"

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
		t.Fatalf("main clean = %v, want true; project=%v", p["clean"], p)
	}
	wts := worktreesOf(p)
	if len(wts) < 1 {
		t.Fatalf("worktrees len = %d, want >= 1; body=%s", len(wts), resp.Body)
	}
	var sawDirty bool
	for _, wt := range wts {
		path, _ := wt["path"].(string)
		name, _ := wt["name"].(string)
		if path == "" || name == "" {
			t.Fatalf("worktree missing path/name: %v", wt)
		}
		if isMain, _ := wt["is_main"].(bool); isMain {
			t.Fatalf("linked worktree is_main=true: %v", wt)
		}
		if clean, ok := wt["clean"].(bool); ok && !clean {
			sawDirty = true
		}
	}
	if !sawDirty {
		t.Fatalf("expected at least one dirty linked worktree; worktrees=%v", wts)
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
