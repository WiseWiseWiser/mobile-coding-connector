# Scenario

**Feature**: local app profile still spawns daemon

```
appprofile.Local() -> SpawnsDaemon=true (local app intent unchanged)
```

## Preconditions

Local product remains daemon-based.

## Steps

1. Set `ProfileName=local`.

## Context

REQUIREMENT leaf: `profile/local`.

```go
import (
	"testing"

	"github.com/xhd2015/doctest/session"
)

func Setup(t *testing.T, _ *session.Doctest, req *Request) error {
	req.ProfileName = "local"
	return nil
}
```
