# Scenario

**Feature**: Publish keeps caller-provided id and ts

```
# non-empty identifiers
Publish(Event{id,ts set}) -> same id and ts
```

## Steps

1. Event with fixed ID and TS.
2. Run hub-publish.

```go
import (
	"encoding/json"
	"testing"

	sharedeb "github.com/xhd2015/dot-pkgs/go-pkgs/eventbus"
	"github.com/xhd2015/doctest/session"
)

func Setup(t *testing.T, d *session.Doctest, req *Request) error {
	t.Helper()
	req.Event = sharedeb.Event{
		ID:      "fixed-id-001",
		TS:      "2026-08-10T12:00:00Z",
		Source:  sharedeb.SourceSeatalkLocalBot,
		Type:    sharedeb.TypeSeatalkMessageReceived,
		Payload: json.RawMessage(`{"msg":"hi"}`),
	}
	return nil
}
```
