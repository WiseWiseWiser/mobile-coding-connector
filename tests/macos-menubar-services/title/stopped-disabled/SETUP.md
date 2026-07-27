# Scenario

**Feature**: stopped disabled service title

```
FormatServiceTitle("api","stopped",false) -> "api ○ Stopped (disabled)"
```

## Preconditions

Service is disabled and not running.

## Steps

1. Set name `api`, status `stopped`, enabled `false`.

## Context

REQUIREMENT leaf: `title/stopped-disabled`.

```go
import (
	"testing"

	"github.com/xhd2015/doctest/session"
)

func Setup(t *testing.T, _ *session.Doctest, req *Request) error {
	req.Name = "api"
	req.Status = "stopped"
	req.Enabled = false
	return nil
}
```