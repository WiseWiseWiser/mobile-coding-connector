# Scenario

**Feature**: port CLI parse / unknown subcommand errors

```
remote-agent port <bad> -> Error: on stderr, non-zero exit
```

## Preconditions

None beyond L2 harness.

## Steps

1. Leaf sets bad Args.
2. Assert non-zero exit and Error: prefix.

## Context

Fatal errors on stderr; non-zero exit.

```go
import (
	"testing"
	"github.com/xhd2015/doctest/session"
)

func Setup(t *testing.T, _ *session.Doctest, req *Request) error {
	req.Op = "cli"
	return nil
}
```
