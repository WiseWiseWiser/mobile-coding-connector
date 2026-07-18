## Expected

1. `Run` completes without error and `Response.Err` is empty.
2. `ModuleRequire` is true — go.mod requires `github.com/xhd2015/bak-files`.
3. `ModuleReplace` is true — go.mod has `replace … => ../..` for bak-files.

## Side Effects

- None (read-only go.mod inspection).

## Errors

- Missing require or replace fails the leaf (expected RED pre-implementer).

## Exit Code

- N/A (in-process).

```go
import "testing"

func Assert(t *testing.T, req *Request, resp *Response, err error) {
	if err != nil {
		t.Fatalf("Run: %v", err)
	}
	if resp.Err != "" {
		t.Fatalf("inspect go.mod: %s", resp.Err)
	}
	if !resp.ModuleRequire {
		t.Fatalf("go.mod missing require %s (detail=%q)", pathflagModule, resp.Detail)
	}
	if !resp.ModuleReplace {
		t.Fatalf("go.mod missing replace %s => ../.. (detail=%q)", pathflagModule, resp.Detail)
	}
}
```
