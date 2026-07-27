# Scenario

**Feature**: default browser preference shows plain label

```
FormatOpenInBrowserLabel("default") -> "Open in Browser"
```

## Preconditions

Default preference selected.

## Steps

1. Set `Browser` to `default`.

## Context

REQUIREMENT leaf: `label/default`.

```go
import (
	"testing"

	"github.com/xhd2015/doctest/session"
)

func Setup(t *testing.T, _ *session.Doctest, req *Request) error {
	req.Browser = "default"
	return nil
}
```