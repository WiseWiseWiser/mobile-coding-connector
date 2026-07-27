# Scenario

**Feature**: default off status title

```
enabled=false / phase=off -> "Status: Off"
```

## Preconditions

Default task state is disabled until the user enables it.

## Steps

1. Enabled=false, Phase=off.

## Context

REQUIREMENT #1 and #11.

```go
import (
	"testing"

	"github.com/xhd2015/doctest/session"
)

func Setup(t *testing.T, _ *session.Doctest, req *Request) error {
	req.Enabled = false
	req.Phase = "off"
	req.NowRFC3339 = "2026-07-10T15:00:00Z"
	return nil
}
```
