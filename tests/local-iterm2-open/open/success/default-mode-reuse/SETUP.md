# Scenario

**Feature**: omitting mode defaults Open to ModeReuseCurrent

```
POST {dir} (no mode) -> Open(..., ModeReuseCurrent) -> 200
```

## Preconditions

Valid temp dir from parent.

## Steps

1. Set `OmitMode=true`.
2. Enable real OpenConfig to capture reuse script markers.

## Context

REQUIREMENT scenario 1: default mode → reuse.

```go
import (
	"testing"
	"github.com/xhd2015/doctest/session"
)
func Setup(t *testing.T, _ *session.Doctest, req *Request) error {
	req.OmitMode = true
	req.UseRealOpenConfig = true
	return nil
}
```
