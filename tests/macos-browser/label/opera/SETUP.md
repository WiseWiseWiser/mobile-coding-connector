# Scenario

**Feature**: Opera preference shows browser suffix

```
FormatOpenInBrowserLabel("opera") -> "Open in Browser(Opera)"
```

## Preconditions

Opera selected in Settings.

## Steps

1. Set `Browser` to `opera`.

## Context

REQUIREMENT leaf: `label/opera`.

```go
import (
	"testing"

	"github.com/xhd2015/doctest/session"
)

func Setup(t *testing.T, _ *session.Doctest, req *Request) error {
	req.Browser = "opera"
	return nil
}
```