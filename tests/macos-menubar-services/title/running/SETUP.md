# Scenario

**Feature**: running service title

```
FormatServiceTitle("web","running",true) -> "web ● Running"
```

## Preconditions

Service is enabled and status is `running`.

## Steps

1. Set name `web`, status `running`, enabled `true`.

## Context

REQUIREMENT leaf: `title/running`.

```go
import (
	"testing"

	"github.com/xhd2015/doctest/session"
)

func Setup(t *testing.T, _ *session.Doctest, req *Request) error {
	req.Name = "web"
	req.Status = "running"
	req.Enabled = true
	return nil
}
```