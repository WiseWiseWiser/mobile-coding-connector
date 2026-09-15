# Scenario

**Feature**: a home without auth.json fails as not authenticated

```
empty home -> status error: not authenticated (no panel, no usage url)
```

## Preconditions

1. The home directory has no `auth.json`.

## Steps

1. Set `Op=no-credentials`.

## Context

A wrong `--home` is the most common Command Code setup mistake, so the error
must name the credential problem instead of a generic failure.

```go
import (
	"testing"

	"github.com/xhd2015/doctest/session"
)
func Setup(t *testing.T, _ *session.Doctest, req *Request) error {
	req.Op = "no-credentials"
	return nil
}
```
