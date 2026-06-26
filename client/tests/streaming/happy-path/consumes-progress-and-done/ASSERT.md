## Expected

1. `err` is nil.
2. Exactly 3 `progress` events received before stream completes.
3. `Response.Done` is non-nil.
4. `Done["healthy"]` is `true`.
5. Event ids in order: `a`, `b`, `c`.

## Side Effects

None.

## Errors

- `StreamErr` non-empty.
- Missing or empty `Done` map.
- Wrong progress count or order.

```go
import "testing"

func Assert(t *testing.T, req *Request, resp *Response, err error) {
	if err != nil {
		t.Fatalf("Run returned unexpected error: %v", err)
	}
	if resp.StreamErr != "" {
		t.Fatalf("Stream returned error: %s", resp.StreamErr)
	}
	var progress int
	var ids []string
	for _, ev := range resp.Events {
		if ev.Type == "progress" {
			progress++
			ids = append(ids, ev.ID)
		}
	}
	if progress != 3 {
		t.Fatalf("got %d progress events, want 3", progress)
	}
	wantIDs := []string{"a", "b", "c"}
	for i, want := range wantIDs {
		if i >= len(ids) || ids[i] != want {
			t.Fatalf("progress ids = %v, want %v", ids, wantIDs)
		}
	}
	if resp.Done == nil {
		t.Fatal("Done map is nil")
	}
	if healthy, ok := resp.Done["healthy"].(bool); !ok || !healthy {
		t.Fatalf("Done[healthy] = %v, want true", resp.Done["healthy"])
	}
}
```
