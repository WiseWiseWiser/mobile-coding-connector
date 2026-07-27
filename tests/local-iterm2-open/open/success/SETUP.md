# Scenario

**Feature**: valid directory open succeeds with correct mode

```
POST open {dir: existingDir, mode?} -> Open called -> 200
```

## Preconditions

Temp directory exists and is a real directory.

## Steps

1. Create temp dir as `Dir`.
2. Leaf sets mode / send / UseRealOpenConfig as needed.

## Context

Success path; mode default reuse when omitted.

```go
import (
	"testing"
	"github.com/xhd2015/doctest/session"
)
func Setup(t *testing.T, _ *session.Doctest, req *Request) error {
	req.Dir = t.TempDir()
	req.OmitSend = true
	return nil
}
```
