## Expected

1. HTTP status `200` (list succeeds overall).
2. Exactly one project entry.
3. Project `error` is a non-empty string (missing path / not git / similar).

## Errors

- Omitting the project entirely when path is missing.
- 5xx instead of reporting per-project error.

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
	errStr, _ := p["error"].(string)
	if errStr == "" {
		t.Fatalf("expected non-empty project error; project=%v body=%s", p, resp.Body)
	}
}
```
