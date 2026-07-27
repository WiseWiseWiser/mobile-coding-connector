# Scenario

**Feature**: parse canonical show-usage output

```
Weekly limit + Next reset lines -> UsageInfo
```

## Preconditions

`show-usage-standard.txt` fixture.

## Steps

1. `FixtureFile=show-usage-standard.txt`, `ExpectParseError=false`.

## Context

REQUIREMENT leaf: `parse/standard-output`.

```go
import (
	"testing"

	"github.com/xhd2015/doctest/session"
)

func Setup(t *testing.T, _ *session.Doctest, req *Request) error {
	req.FixtureFile = "show-usage-standard.txt"
	req.ExpectParseError = false
	return nil
}
```