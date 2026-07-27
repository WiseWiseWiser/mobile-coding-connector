# Scenario

**Feature**: unknown browser value falls back to plain label

```
FormatOpenInBrowserLabel("safari") -> "Open in Browser"
```

## Preconditions

Stored value is not one of the supported preferences.

## Steps

1. Set `Browser` to `safari`.

## Context

REQUIREMENT leaf: `label/unknown`.

```go
import (
	"testing"

	"github.com/xhd2015/doctest/session"
)

func Setup(t *testing.T, _ *session.Doctest, req *Request) error {
	req.Browser = "safari"
	return nil
}
```