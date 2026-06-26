## Expected

1. `len(Response.Events)` is 4.
2. Event types in order: `progress`, `progress`, `section`, `done`.
3. First progress frame decodes with `id: step_a`, `status: ok`.
4. Second progress frame decodes with `id: step_b`, `status: ok`.
5. Section frame decodes with `type: section` and non-empty `message`.
6. Done frame decodes with `healthy: true`.

## Side Effects

None — in-memory `httptest.ResponseRecorder` only.

## Errors

- Missing or reordered frames.
- Done frame missing `healthy: true`.

```go
import (
	"testing"
)

func Assert(t *testing.T, req *Request, resp *Response, err error) {
	if err != nil {
		t.Fatalf("Run returned unexpected error: %v", err)
	}
	if len(resp.Events) != 4 {
		t.Fatalf("got %d SSE events, want 4; types=%v", len(resp.Events), eventTypes(resp.Events))
	}
	wantTypes := []string{"progress", "progress", "section", "done"}
	for i, want := range wantTypes {
		if resp.Events[i].Type != want {
			t.Fatalf("event[%d].Type = %q, want %q (all types: %v)", i, resp.Events[i].Type, want, eventTypes(resp.Events))
		}
	}
	if id, _ := resp.Events[0].Decoded["id"].(string); id != "step_a" {
		t.Fatalf("first progress id = %q, want step_a", id)
	}
	if id, _ := resp.Events[1].Decoded["id"].(string); id != "step_b" {
		t.Fatalf("second progress id = %q, want step_b", id)
	}
	if resp.DoneHealthy == nil || !*resp.DoneHealthy {
		t.Fatalf("done.healthy = %v, want true", resp.DoneHealthy)
	}
}
```
