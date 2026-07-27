# Scenario

**Feature**: parse error when next reset line missing

```
stdout without Next reset -> error
```

## Preconditions

`show-usage-missing-reset.txt`.

## Steps

1. `ExpectParseError=true`.

## Context

REQUIREMENT leaf: `parse/missing-reset`.

```go
import (
	"testing"

	"github.com/xhd2015/doctest/session"
)

func Setup(t *testing.T, _ *session.Doctest, req *Request) error {
	req.FixtureFile = "show-usage-missing-reset.txt"
	req.ExpectParseError = true
	return nil
}
```