# Scenario

**Feature**: BuildSSHTunnelDial returns non-nil Dial when Client provided

```
# wiring unit
agentcli.BuildSSHTunnelDial(c, pub) -> dial != nil, session_id set
dial() smoke opens conn
```

## Preconditions

- Scenario: `serve-wiring-dial-from-client`.

## Steps

1. Call BuildSSHTunnelDial with live httptest Client.
2. Assert DialIsNil false; WiringErr empty; WiringSessID non-empty.

## Context

- Requirement scenario: serve starter with Client builds non-nil Dial.

```go
import (
	"testing"

	"github.com/xhd2015/doctest/session"
)

func Setup(t *testing.T, d *session.Doctest, req *Request) error {
	t.Helper()
	_ = d
	req.Scenario = ScenarioServeWiringDial
	return nil
}
```
