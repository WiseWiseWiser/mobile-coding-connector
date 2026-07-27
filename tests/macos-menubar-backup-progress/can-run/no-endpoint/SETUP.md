# Scenario

**Feature**: Backup Now blocked without endpoint

```
CanRunBackupNow(hasEndpoint=false, running=false, server="foo.example.com") -> false
```

## Preconditions

No remote endpoint configured.

## Steps

1. HasEndpoint=false; otherwise ready.

## Context

REQUIREMENT #3.

```go
import (
	"testing"

	"github.com/xhd2015/doctest/session"
)

func Setup(t *testing.T, _ *session.Doctest, req *Request) error {
	req.HasEndpoint = false
	req.Running = false
	req.ServerName = "foo.example.com"
	return nil
}
```
