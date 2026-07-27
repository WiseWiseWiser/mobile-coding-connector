# Scenario

**Feature**: remote-agent config --help

```
# --help -> same help family as bare
remote-agent config --help -> stdout help, exit 0
```

## Preconditions

None beyond remote profile.

## Steps

1. Args = `config --help`.

## Context

T2 long form.

```go
import (
	"testing"

	"github.com/xhd2015/doctest/session"
)

func Setup(t *testing.T, _ *session.Doctest, req *Request) error {
	req.Args = []string{"config", "--help"}
	return nil
}
```
