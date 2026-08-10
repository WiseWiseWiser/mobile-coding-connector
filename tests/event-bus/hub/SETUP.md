# Scenario

**Feature**: in-memory Hub publish / ring / fan-out

```
# Hub is the core bus
NewHub(ring) -> Publish | Subscribe | Recent
```

## Preconditions

`Op` is a hub surface (`hub-publish` or `hub-subscribe`).

## Steps

1. Default Op to hub-publish; fanout group overrides.
2. Leaves set Event / PublishCount / SubscriberCount.

## Context

REQUIREMENT scenarios 1–2.

```go
import (
	"testing"

	"github.com/xhd2015/doctest/session"
)

func Setup(t *testing.T, d *session.Doctest, req *Request) error {
	t.Helper()
	if req.Op == "" {
		req.Op = "hub-publish"
	}
	return nil
}
```
