# Scenario

**Feature**: Hub ring buffer capacity

```
# ring keeps last N events
Publish x M (M>N) -> Recent(N) length N newest
```

## Preconditions

`Op=hub-publish` with PublishCount and RingSize.

## Steps

1. Set Op hub-publish.
2. Leaf sets ring size and publish count.

## Context

REQUIREMENT scenario 1 (ring ~200).

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
