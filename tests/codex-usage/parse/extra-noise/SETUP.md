# Scenario

**Feature**: parse usage lines buried in scrollback noise

```
noisy stdout -> ParseStatusOutput -> UsageInfo
```

## Preconditions

`show-status-noisy.txt` fixture.

## Steps

1. `FixtureFile=show-status-noisy.txt`.

## Context

REQUIREMENT leaf: `parse/extra-noise`.

```go
import (
	"testing"

	"github.com/xhd2015/doctest/session"
)

func Setup(t *testing.T, _ *session.Doctest, req *Request) error {
	req.FixtureFile = "show-status-noisy.txt"
	req.ExpectParseError = false
	return nil
}
```