---
label: slow && negative && requires-dist
explanation: builds keep-alive daemon (needs ai-critic-react/dist); polls until error JSON cached
---

## Expected

Daemon must surface fetch failure as error JSON (not stuck loading):

1. `APIStatusCode` is `200`.
2. `APIParsed.Status` is `error`.
3. `APIParsed.Error` contains `timeout waiting for status output`.
4. `APIParsed.UpdatedAt` is non-empty.

## Errors

- `status=ready` without parseable TUI output.
- API stuck in `loading` after wait window.

```go
import (
	"strings"
	"testing"
)

func Assert(t *testing.T, req *Request, resp *Response, err error) {
	if err != nil {
		t.Fatal(err)
	}
	if resp.APIStatusCode != 200 {
		t.Fatalf("status code = %d body=%s", resp.APIStatusCode, resp.APIBody)
	}
	if resp.APIParsed == nil {
		t.Fatalf("could not parse API body: %s", resp.APIBody)
	}
	if resp.APIParsed.Status != "error" {
		t.Fatalf("json status = %q, want error; body=%s", resp.APIParsed.Status, resp.APIBody)
	}
	if !strings.Contains(resp.APIParsed.Error, "timeout waiting for status output") {
		t.Fatalf("json error = %q, want timeout waiting for status output", resp.APIParsed.Error)
	}
	if resp.APIParsed.UpdatedAt == "" {
		t.Fatal("updated_at missing in API error response")
	}
}
```