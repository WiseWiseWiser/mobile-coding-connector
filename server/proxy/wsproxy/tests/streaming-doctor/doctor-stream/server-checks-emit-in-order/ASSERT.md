## Expected

1. First event is `meta` or `section` (doctor header/banner).
2. ≥5 `progress` events with `layer: server`.
3. Each server `progress.id` appears exactly once.
4. Server progress ids appear in the same order as `expectedServerDoctorCheckOrder()`
   (subset match — only ids present in the stream are compared in order).
5. Final event is `type: done`.
6. `done.healthy` is `false` (tunnel mapping absent in this fixture).
7. No server check appears only inside `done` — all are individual `progress` events.

## Side Effects

None.

## Errors

- Stream ends without `done`.
- Duplicate progress ids.
- Progress ids out of order relative to legacy `serverDoctorChecks`.

```go
import (
	"testing"
)

func Assert(t *testing.T, req *Request, resp *Response, err error) {
	if err != nil {
		t.Fatalf("Run returned unexpected error: %v", err)
	}
	if len(resp.Events) == 0 {
		t.Fatal("no SSE events captured")
	}
	first := resp.Events[0].Type
	if first != "meta" && first != "section" {
		t.Fatalf("first event type = %q, want meta or section", first)
	}
	last := resp.Events[len(resp.Events)-1]
	if last.Type != "done" {
		t.Fatalf("final event type = %q, want done", last.Type)
	}

	serverProgress := serverProgressLayer(resp.Events)
	if len(serverProgress) < 5 {
		t.Fatalf("got %d server progress events, want at least 5", len(serverProgress))
	}

	seen := make(map[string]int)
	var ids []string
	for _, ev := range serverProgress {
		id, _ := ev.Decoded["id"].(string)
		if id == "" {
			t.Fatal("server progress event missing id")
		}
		seen[id]++
		if seen[id] > 1 {
			t.Fatalf("duplicate progress id %q", id)
		}
		ids = append(ids, id)
	}

	wantOrder := expectedServerDoctorCheckOrder()
	wantIdx := 0
	for _, id := range ids {
		for wantIdx < len(wantOrder) && wantOrder[wantIdx] != id {
			wantIdx++
		}
		if wantIdx >= len(wantOrder) {
			t.Fatalf("progress id %q not found in expected order %v (got %v)", id, wantOrder, ids)
		}
		wantIdx++
	}

	if resp.DoneHealthy == nil || *resp.DoneHealthy {
		t.Fatalf("done.healthy = %v, want false (tunnel mapping absent)", resp.DoneHealthy)
	}
}
```
