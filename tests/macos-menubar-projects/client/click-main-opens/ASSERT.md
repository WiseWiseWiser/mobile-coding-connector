## Expected

1. `ClickMainOpensProjectPath` is true — sources include:
   - a primary **“Open in iTerm2”** (or “Open in iTerm”) control under the project submenu
   - `openITerm2` / local open API usage
   - open of `project.path` with reuse semantics (explicit or default)

## Errors

- Empty project Menu with no open action.
- Relying only on Menu title click (not a valid UX target).

```go
import "testing"

func Assert(t *testing.T, req *Request, resp *Response, err error) {
	if err != nil {
		t.Fatal(err)
	}
	if !resp.ClickMainOpensProjectPath {
		t.Fatalf("project submenu must have Open in iTerm2 opening project.path via openITerm2 reuse (sources: %v)", resp.SwiftSourcesChecked)
	}
}
```
