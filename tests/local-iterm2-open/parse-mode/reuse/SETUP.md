# Scenario

**Feature**: mode "reuse" maps to ModeReuseCurrent

```
ParseOpenMode("reuse") -> ModeReuseCurrent
```

## Preconditions

None beyond parse group.

## Steps

1. Set `ModeInput=reuse`.

## Context

JSON `"reuse"` → shell/iterm2 `ModeReuseCurrent` (kool `-r`).

```go
import (
	"testing"
	"github.com/xhd2015/doctest/session"
)
func Setup(t *testing.T, _ *session.Doctest, req *Request) error {
	req.ModeInput = "reuse"
	return nil
}
```
