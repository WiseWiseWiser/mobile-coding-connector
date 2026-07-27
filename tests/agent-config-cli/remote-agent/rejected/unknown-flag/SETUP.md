# Scenario

**Feature**: unknown config flag is rejected

```
# unknown flag -> non-zero error (point at help)
remote-agent config --not-a-real-flag -> error
```

## Preconditions

None.

## Steps

1. Args = `config --not-a-real-flag`.

## Context

T8.

```go
import (
	"testing"

	"github.com/xhd2015/doctest/session"
)

func Setup(t *testing.T, _ *session.Doctest, req *Request) error {
	req.Args = []string{"config", "--not-a-real-flag"}
	return nil
}
```
