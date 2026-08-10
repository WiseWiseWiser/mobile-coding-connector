# Scenario

**Feature**: two subscribers both receive the published event

```
# fan-out
Subscribe, Subscribe -> Publish(ev) -> both receive ev (with id/ts)
```

## Steps

1. SubscriberCount=2.
2. Fixture event with known type/source.

```go
import (
	"encoding/json"
	"testing"

	sharedeb "github.com/xhd2015/dot-pkgs/go-pkgs/eventbus"
	"github.com/xhd2015/doctest/session"
)

func Setup(t *testing.T, d *session.Doctest, req *Request) error {
	t.Helper()
	req.SubscriberCount = 2
	req.Event = sharedeb.Event{
		Source:  sharedeb.SourceAgentRun,
		Type:    sharedeb.TypeAgentTTYStarted,
		Payload: json.RawMessage(`{"session":"s1"}`),
	}
	return nil
}
```
