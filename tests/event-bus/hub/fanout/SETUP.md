# Scenario

**Feature**: Hub fan-out to live subscribers

```
# multiple Subscribe channels receive each Publish
Subscribe x2 -> Publish -> both channels deliver Event
```

## Preconditions

`Op=hub-subscribe`.

## Steps

1. Set Op hub-subscribe.
2. Leaf sets SubscriberCount and Event.

## Context

REQUIREMENT scenario 2.

```go
import (
	"testing"

	"github.com/xhd2015/doctest/session"
)

func Setup(t *testing.T, d *session.Doctest, req *Request) error {
	t.Helper()
	req.Op = "hub-subscribe"
	return nil
}
```
