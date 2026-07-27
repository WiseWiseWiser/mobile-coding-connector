# Scenario

**Feature**: open endpoint is not auth skip-listed

```
server.Serve auth.Middleware skip slice -> must not contain /api/local/iterm2/open
```

## Preconditions

`server/server.go` exists (host registration site).

## Steps

1. Set `Op=skip_list`.

## Context

REQUIREMENT: Bearer required (not skip-listed).

```go
import (
	"testing"
	"github.com/xhd2015/doctest/session"
)
func Setup(t *testing.T, _ *session.Doctest, req *Request) error {
	req.Op = "skip_list"
	return nil
}
```
