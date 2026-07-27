# Scenario

**Feature**: parse usage lines buried in scrollback noise

```
noisy stdout -> ParseShowUsageOutput -> UsageInfo
```

## Preconditions

`show-usage-noisy.txt` fixture.

## Steps

1. `FixtureFile=show-usage-noisy.txt`.

## Context

REQUIREMENT leaf: `parse/extra-noise`.

```go
import (
	"testing"

	"github.com/xhd2015/doctest/session"
)

func Setup(t *testing.T, _ *session.Doctest, req *Request) error {
	req.FixtureFile = "show-usage-noisy.txt"
	req.ExpectParseError = false
	return nil
}
```