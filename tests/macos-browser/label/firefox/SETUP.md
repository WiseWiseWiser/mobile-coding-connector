# Scenario

**Feature**: Firefox preference shows browser suffix

```
FormatOpenInBrowserLabel("firefox") -> "Open in Browser(Firefox)"
```

## Preconditions

Firefox selected in Settings.

## Steps

1. Set `Browser` to `firefox`.

## Context

REQUIREMENT leaf: `label/firefox`.

```go
import (
	"testing"

	"github.com/xhd2015/doctest/session"
)

func Setup(t *testing.T, _ *session.Doctest, req *Request) error {
	req.Browser = "firefox"
	return nil
}
```