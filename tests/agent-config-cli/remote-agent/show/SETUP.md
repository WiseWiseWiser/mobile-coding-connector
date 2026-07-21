# Scenario

**Feature**: remote-agent config --show

```
# --show loads remote-agent-config.json and prints pretty JSON
remote-agent config --show [--json] -> stdout pretty agentConfig
```

## Preconditions

Optional seed under isolated HOME.

## Steps

1. Child leaves set seed and `--show` / `--json` args.

## Context

T3–T5.

```go
import "testing"

func Setup(t *testing.T, req *Request) error {
	// Group default: --show path; leaves may append --json or set seed.
	if len(req.Args) == 0 {
		req.Args = []string{"config", "--show"}
	}
	return nil
}
```
