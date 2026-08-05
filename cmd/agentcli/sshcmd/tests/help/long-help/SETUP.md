# Scenario

**Feature**: long help flag prints Usage

```
# long form
operator -> sshcmd with args ["--help"]
  -> Usage on stdout ending with \n; mentions --serve and [user@host]
```

## Preconditions

- Args: `["--help"]` only.
- No active session required.

## Steps

1. Set `Args` to `["--help"]`.
2. Run Parse + Run; Assert checks stdout and nil errors.

## Context

- Long-form help flag.

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
