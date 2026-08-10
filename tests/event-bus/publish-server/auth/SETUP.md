# Scenario

**Feature**: PublishServer Bearer token policy

```
# open when Token empty; otherwise require Authorization: Bearer
POST /publish + auth headers -> 2xx | 401
```

## Preconditions

`Op=publish-http`.

## Steps

1. Set Op publish-http.
2. Leaf configures ServerToken / ClientToken / OmitAuth.

## Context

REQUIREMENT scenarios 3–4.

```go
import (
	"encoding/json"
	"testing"

	sharedeb "github.com/xhd2015/dot-pkgs/go-pkgs/eventbus"
	"github.com/xhd2015/doctest/session"
)

func Setup(t *testing.T, d *session.Doctest, req *Request) error {
	t.Helper()
	req.Op = "publish-http"
	if req.Event.Type == "" {
		req.Event = sharedeb.Event{
			Source:  sharedeb.SourceAgentRun,
			Type:    sharedeb.TypeAgentTTYStarted,
			Payload: json.RawMessage(`{"auth":"probe"}`),
		}
	}
	return nil
}
```
