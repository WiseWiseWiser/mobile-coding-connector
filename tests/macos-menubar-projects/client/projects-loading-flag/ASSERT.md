## Expected

1. `HasProjectsLoadingFlag` is true — `AppState` (or equivalent) declares
   `projectsLoading`.
2. `KeepsProjectsOnRefreshStart` is true — no `projects = []` wipe on refresh start.
3. `KeepsProjectsOnRefreshFail` is true — failure path keeps prior list (same
   no-wipe contract).

## Side Effects

- None (read-only source inspection).

## Errors

- Missing loading flag; clearing the array when refresh begins or errors.

```go
import "testing"

func Assert(t *testing.T, req *Request, resp *Response, err error) {
	if err != nil {
		t.Fatal(err)
	}
	if !resp.HasProjectsLoadingFlag {
		t.Fatalf("projectsLoading not found in Swift sources: %v", resp.SwiftSourcesChecked)
	}
	if !resp.KeepsProjectsOnRefreshStart {
		t.Fatal("expected projects to be kept on refresh start (no projects = [] wipe)")
	}
	if !resp.KeepsProjectsOnRefreshFail {
		t.Fatal("expected projects to be kept on refresh failure")
	}
}
```
