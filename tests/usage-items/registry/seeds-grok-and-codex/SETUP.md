# Scenario

**Feature**: first run seeds grok and codex as rotating items

```
no registry file -> Seed() -> grok + codex, default grok, rotate true
```

## Preconditions

1. No registry file exists yet (the leaf writes no seed).

## Steps

1. Set `Op=registry`.

## Context

This is today's menu bar: the two providers alternate every 60 seconds.

```go
import (
	"testing"

	"github.com/xhd2015/doctest/session"
)
func Setup(t *testing.T, _ *session.Doctest, req *Request) error {
	req.Op = "registry"
	return nil
}
```
