## Expected

1. `config_load` appears in `Response.ProgressIDs` before `upstream_fetch`.
2. `upstream_proxy` appears before `upstream_fetch`.
3. Index of `config_load` is strictly less than index of `upstream_fetch`.
4. Index of `upstream_proxy` is strictly less than index of `upstream_fetch`.
5. Stream still ends with `done`.

## Side Effects

Test hook `SetTestUpstreamFetchDelay` must be cleared on cleanup.

## Errors

- `upstream_fetch` is the first server progress event (buffered-until-end bug).
- `config_load` or `upstream_proxy` missing from stream.

```go
import (
	"testing"
)

func indexOf(ids []string, target string) int {
	for i, id := range ids {
		if id == target {
			return i
		}
	}
	return -1
}

func Assert(t *testing.T, req *Request, resp *Response, err error) {
	if err != nil {
		t.Fatalf("Run returned unexpected error: %v", err)
	}
	cfgIdx := indexOf(resp.ProgressIDs, "config_load")
	proxyIdx := indexOf(resp.ProgressIDs, "upstream_proxy")
	fetchIdx := indexOf(resp.ProgressIDs, "upstream_fetch")
	if cfgIdx < 0 {
		t.Fatal("missing progress id config_load")
	}
	if proxyIdx < 0 {
		t.Fatal("missing progress id upstream_proxy")
	}
	if fetchIdx < 0 {
		t.Fatal("missing progress id upstream_fetch")
	}
	if cfgIdx >= fetchIdx {
		t.Fatalf("config_load index %d must be < upstream_fetch index %d", cfgIdx, fetchIdx)
	}
	if proxyIdx >= fetchIdx {
		t.Fatalf("upstream_proxy index %d must be < upstream_fetch index %d", proxyIdx, fetchIdx)
	}
	if resp.Events[len(resp.Events)-1].Type != "done" {
		t.Fatalf("final event = %q, want done", resp.Events[len(resp.Events)-1].Type)
	}
}
```
