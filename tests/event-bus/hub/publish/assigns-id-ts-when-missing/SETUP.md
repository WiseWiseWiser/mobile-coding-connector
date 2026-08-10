# Scenario

**Feature**: Publish fills empty id and ts

```
# empty identifiers
Publish(Event{id:"",ts:""}) -> id non-empty, ts non-empty
```

## Steps

1. Event with empty ID and TS, valid source/type/payload.
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
		ID:      "",
		TS:      "",
		Source:  sharedeb.SourceAgentRun,
		Type:    sharedeb.TypeAgentTTYStarted,
		Payload: json.RawMessage(`{"ok":true}`),
	}
	return nil
}
```
