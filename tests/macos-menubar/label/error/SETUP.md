# Scenario

**Bug**: menu bar must show short fixed grok error label

```
FormatGrokLabel("error","","timeout waiting") -> "Grok err"
```

## Preconditions

Daemon surfaces fetch timeout in `error` field; menu bar hides full message.

## Steps

1. Use requirement table message text.

## Context

REQUIREMENT leaf: `label/error` (amended). Full message appears in dropdown only.

```go
import (
	"testing"

	"github.com/xhd2015/doctest/session"
)

func Setup(t *testing.T, _ *session.Doctest, req *Request) error {
	req.Status = "error"
	req.WeeklyLimit = ""
	req.ErrorMsg = "timeout waiting for usage"
	return nil
}
```