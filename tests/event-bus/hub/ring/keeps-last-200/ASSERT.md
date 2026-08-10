## Expected

1. No error.
2. `len(Recent) == 200` (not 250).
3. Chronological order: first retained corresponds to publish index 50 payload
   `{"n":50}` (or last event is n=249) when payload encodes sequence.

## Errors

- Ring larger than 200 or empty.
- Wrong retention order (newest dropped).

```go
import (
	"encoding/json"
	"testing"

	"github.com/xhd2015/doctest/session"
)

func Assert(t *testing.T, d *session.Doctest, req *Request, resp *Response, err error) {
	t.Helper()
	if err != nil {
		t.Fatalf("Run error: %v", err)
	}
	if len(resp.Recent) != 200 {
		t.Fatalf("Recent len = %d, want 200", len(resp.Recent))
	}
	// Newest last: last payload n should be 249
	var last struct {
		N int `json:"n"`
	}
	if err := json.Unmarshal(resp.Recent[len(resp.Recent)-1].Payload, &last); err != nil {
		t.Fatalf("unmarshal last payload: %v", err)
	}
	if last.N != 249 {
		t.Fatalf("last event n = %d, want 249", last.N)
	}
	var first struct {
		N int `json:"n"`
	}
	if err := json.Unmarshal(resp.Recent[0].Payload, &first); err != nil {
		t.Fatalf("unmarshal first payload: %v", err)
	}
	if first.N != 50 {
		t.Fatalf("first retained n = %d, want 50 (dropped 0..49)", first.N)
	}
	for _, ev := range resp.Recent {
		if ev.ID == "" || ev.TS == "" {
			t.Fatalf("ring event missing id/ts: %+v", ev)
		}
	}
}
```
