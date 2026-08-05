# Scenario

**Feature**: After Close, further dials to the former LocalPort fail

```
# teardown
LocalRelay.Start -> Close
client dial 127.0.0.1:formerPort -> connection error
```

## Preconditions

- Scenario: `relay-close`.

## Steps

1. Start LocalRelay; record LocalPort.
2. Close relay.
3. Dial former port; Assert dial fails (DialAfterCloseErr non-empty).

## Context

- Ensures Close stops the listener (not only ignores new accepts quietly).

```go
import (
	"testing"

	"github.com/xhd2015/doctest/session"
)

func Setup(t *testing.T, d *session.Doctest, req *Request) error {
	t.Helper()
	_ = d
	req.Scenario = ScenarioRelayClose
	return nil
}
```
