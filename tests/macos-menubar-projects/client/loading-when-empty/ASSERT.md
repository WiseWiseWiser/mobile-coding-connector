## Expected

1. `ShowsLoadingWhenEmpty` is true — sources reference `Loading…` (or
   `formatProjectsLoadingLabel`) **and** gate on `projectsLoading` with empty list.

## Side Effects

- None (read-only source inspection).

## Errors

- Always shows `No wrk projects` while first load is still in flight.

```go
import "testing"

func Assert(t *testing.T, req *Request, resp *Response, err error) {
	if err != nil {
		t.Fatal(err)
	}
	if !resp.ShowsLoadingWhenEmpty {
		t.Fatalf("expected Loading… when projectsLoading && empty; sources: %v", resp.SwiftSourcesChecked)
	}
}
```
