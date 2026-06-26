## Expected

1. `err` is nil; `StreamErr` is empty.
2. Two events with `Type == "log"`.
3. First log `Message == "starting"`; second `Message == "ready"`.
4. `Done["healthy"]` is `true`.

## Side Effects

None.

## Errors

- Log events decoded as wrong type.
- Empty `Message` on log events.

```go
import (
	"testing"

	"github.com/xhd2015/ai-critic/client"
)

func Assert(t *testing.T, req *Request, resp *Response, err error) {
	if err != nil {
		t.Fatalf("Run returned unexpected error: %v", err)
	}
	if resp.StreamErr != "" {
		t.Fatalf("Stream returned error: %s", resp.StreamErr)
	}
	var logs []client.StreamEvent
	for _, ev := range resp.Events {
		if ev.Type == "log" {
			logs = append(logs, ev)
		}
	}
	if len(logs) != 2 {
		t.Fatalf("got %d log events, want 2 (all: %+v)", len(logs), resp.Events)
	}
	if logs[0].Message != "starting" || logs[1].Message != "ready" {
		t.Fatalf("log messages = %q, %q; want starting, ready", logs[0].Message, logs[1].Message)
	}
}
```
