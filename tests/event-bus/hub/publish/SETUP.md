# Scenario

**Feature**: Hub.Publish id/ts enrichment rules

```
# missing id/ts filled; provided values preserved
Hub.Publish(Event) -> Event with id+ts policy
```

## Preconditions

`Op=hub-publish` with a single Event fixture.

## Steps

1. Set Op to hub-publish.
2. Leaf sets Event id/ts source/type/payload.

## Context

REQUIREMENT scenario 1 (assign / preserve).

```go
import (
	"testing"

	"github.com/xhd2015/doctest/session"
)

func Setup(t *testing.T, d *session.Doctest, req *Request) error {
	t.Helper()
	req.Op = "hub-publish"
	return nil
}
```
