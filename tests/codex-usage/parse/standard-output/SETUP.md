# Scenario

**Feature**: parse canonical show-status output

```
Monthly usage + Credits used + Next reset lines -> UsageInfo
```

## Preconditions

`show-status-standard.txt` fixture.

## Steps

1. `FixtureFile=show-status-standard.txt`, `ExpectParseError=false`.

## Context

REQUIREMENT leaf: `parse/standard-output`.

```go
import (
	"testing"

	"github.com/xhd2015/doctest/session"
)

func Setup(t *testing.T, _ *session.Doctest, req *Request) error {
	req.FixtureFile = "show-status-standard.txt"
	req.ExpectParseError = false
	return nil
}
```