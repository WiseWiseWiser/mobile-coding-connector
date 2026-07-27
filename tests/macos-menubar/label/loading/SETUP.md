# Scenario

**Feature**: loading status shows placeholder

```
FormatGrokLabel("loading","","") -> "Grok ..."
```

## Preconditions

Status loading; no limit or error yet.

## Steps

1. Empty weekly limit and error message.

## Context

REQUIREMENT leaf: `label/loading`.

```go
import (
	"testing"

	"github.com/xhd2015/doctest/session"
)

func Setup(t *testing.T, _ *session.Doctest, req *Request) error {
	req.Status = "loading"
	req.WeeklyLimit = ""
	req.ErrorMsg = ""
	return nil
}
```