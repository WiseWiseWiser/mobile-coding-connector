# Scenario

**Feature**: port root help

```
remote-agent port --help -> Usage: remote-agent port ...
```

## Preconditions

None.

## Steps

1. Args = `port --help` (via global help on subcommand path).

## Context

Scenario 1 help at port level.

```go
import (
	"testing"
	"github.com/xhd2015/doctest/session"
)

func Setup(t *testing.T, _ *session.Doctest, req *Request) error {
	setCLI(req, "port", "--help")
	return nil
}
```
