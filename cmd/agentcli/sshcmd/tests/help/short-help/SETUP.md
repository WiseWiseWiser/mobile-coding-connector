# Scenario

**Feature**: short help flag prints Usage

```
# short form
operator -> sshcmd with args ["-h"]
  -> same help contract as --help
```

## Preconditions

- Args: `["-h"]` only.

## Steps

1. Set `Args` to `["-h"]`.
2. Run Parse + Run; Assert same help outcomes as long form.

## Context

- Short-form help flag.

```go
import (
	"testing"

	"github.com/xhd2015/doctest/session"
)

func Setup(t *testing.T, d *session.Doctest, req *Request) error {
	t.Helper()
	_ = d
	req.Args = []string{"-h"}
	return nil
}
```
