# Scenario

**Feature**: unison --help prints unison help

```
operator -> RunCLI([unison, --help]) -> unison Usage
```

## Preconditions

- Args: `["unison", "--help"]`.

## Steps

1. Set Args.
2. Assert unison help verbs.

## Context

- Long help under unison.

```go
import (
	"testing"

	"github.com/xhd2015/doctest/session"
)

func Setup(t *testing.T, d *session.Doctest, req *Request) error {
	t.Helper()
	_ = d
	req.Args = []string{"unison", "--help"}
	return nil
}
```
