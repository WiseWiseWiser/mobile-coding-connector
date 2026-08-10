# Scenario

**Feature**: main mux WebSocket subscribe path

```
# RegisterSubscribeWS fans hub events to WS clients
RegisterSubscribeWS(mux, hub) -> GET /api/event-bus/ws
  <- JSON Event after Hub.Publish or HTTP /publish
```

## Preconditions

`Op=subscribe-ws`. Real loopback HTTP server (WS upgrade).

## Steps

1. Set Op subscribe-ws.
2. Leaf chooses PublishVia hub|http and Event fixture.

## Context

REQUIREMENT scenario 6.

```go
import (
	"encoding/json"
	"testing"

	sharedeb "github.com/xhd2015/dot-pkgs/go-pkgs/eventbus"
	"github.com/xhd2015/doctest/session"
)

func Setup(t *testing.T, d *session.Doctest, req *Request) error {
	t.Helper()
	req.Op = "subscribe-ws"
	if req.Event.Type == "" {
		req.Event = sharedeb.Event{
			Source:  sharedeb.SourceSeatalkLocalBot,
			Type:    sharedeb.TypeSeatalkMessageReceived,
			Payload: json.RawMessage(`{"text":"ws-probe"}`),
		}
	}
	return nil
}
```
