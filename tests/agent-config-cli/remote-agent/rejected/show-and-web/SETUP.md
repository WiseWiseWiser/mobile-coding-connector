# Scenario

**Feature**: --show and --web are mutually exclusive

```
# both flags -> non-zero mutual exclusion error
remote-agent config --show --web -> error
```

## Preconditions

None.

## Steps

1. Args = `config --show --web`.

## Context

T7.

```go
import (
	"testing"

	"github.com/xhd2015/doctest/session"
)

func Setup(t *testing.T, _ *session.Doctest, req *Request) error {
	req.Args = []string{"config", "--show", "--web"}
	return nil
}
```
