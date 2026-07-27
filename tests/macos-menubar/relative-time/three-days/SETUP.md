# Scenario

**Feature**: grok reset ≥24h away → days unit

```
FormatTimeLeft("July 9, 16:55 PT", now=Jul 6 16:55 PDT) -> "left 3d"
```

## Preconditions

Grok reset string with PT timezone suffix.

## Steps

1. Set reset and fixed `now` per requirement table.

## Context

REQUIREMENT leaf: `relative-time/three-days`.

```go
import (
	"testing"

	"github.com/xhd2015/doctest/session"
)

func Setup(t *testing.T, _ *session.Doctest, req *Request) error {
	req.Reset = "July 9, 16:55 PT"
	req.NowRFC3339 = "2026-07-06T16:55:00-07:00"
	return nil
}
```