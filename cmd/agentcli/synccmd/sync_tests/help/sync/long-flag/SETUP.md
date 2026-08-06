# Scenario

**Feature**: sync --help prints Usage

```
operator -> RunCLI([--help]) -> sync Usage
```

## Preconditions

- Args: `["--help"]`.

## Steps

1. Set Args to `--help`.
2. Assert same help contract as bare.

## Context

- Long help flag.

```go
import (
	"testing"

	"github.com/xhd2015/doctest/session"
)

func Setup(t *testing.T, d *session.Doctest, req *Request) error {
	t.Helper()
	_ = d
	req.Args = []string{"--help"}
	return nil
}
```
