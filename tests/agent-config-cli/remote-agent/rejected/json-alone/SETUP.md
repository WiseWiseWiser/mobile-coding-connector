# Scenario

**Feature**: --json without --show is an error

```
# --json alone requires --show
remote-agent config --json -> non-zero, message mentions --show
```

## Preconditions

None.

## Steps

1. Args = `config --json`.

## Context

T6.

```go
import (
	"testing"

	"github.com/xhd2015/doctest/session"
)

func Setup(t *testing.T, _ *session.Doctest, req *Request) error {
	req.Args = []string{"config", "--json"}
	return nil
}
```
