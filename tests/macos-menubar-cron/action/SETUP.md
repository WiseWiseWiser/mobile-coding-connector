# Scenario

**Feature**: per-cron-task action gating (Run Now + Enable/Disable)

```
status + enabled -> CanRunCronTask / ShowEnableCronAction -> booleans
```

## Preconditions

`Op=action` dispatches to action gating helpers in `macosapp/menubar`.

## Steps

1. Leaf supplies `Status` and/or `Enabled`.

## Context

REQUIREMENT: disable Run Now when running; toggle Enable/Disable by enabled.

```go
import (
	"testing"

	"github.com/xhd2015/doctest/session"
)

func Setup(t *testing.T, _ *session.Doctest, req *Request) error {
	req.Op = "action"
	return nil
}
```
