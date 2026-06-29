# Scenario

**Feature**: CLI rejects invalid `git` invocations before HTTP

```
# remote-agent parses git argv locally
remote-agent git <bad argv> -> Error on stderr, no POST /api/remote-agent/git/run
```

## Preconditions

Server is running (harness still starts it); CLI must fail before any `/run` call.

## Steps

1. Leaf sets `Request.Args` to an invalid `git` argv sequence.
2. `Run` executes remote-agent and captures stderr.

## Context

Covers missing `-C` and unknown subcommand per requirement.

```go
import (
	"testing"

	"github.com/xhd2015/ai-critic/script/lib"
)

func Setup(t *testing.T, req *Request) error {
	if req.Token == "" {
		req.Token = lib.TestPassword
	}
	return nil
}
```