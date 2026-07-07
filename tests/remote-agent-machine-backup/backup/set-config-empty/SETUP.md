# Scenario

**Feature**: bare --set-config rejects missing input

```
# set-config without --exclude or --large-dir-threshold -> CLI error
remote-agent machine backup --set-config -> non-zero exit
```

## Preconditions

Default `serverHome` fixtures.

## Steps

1. `SetConfig=true` with no excludes or threshold.
2. Args: `machine backup`.

## Context

Validation leaf: set-config requires at least one input flag.

```go
import "testing"

func Setup(t *testing.T, req *Request) error {
	req.SetConfig = true
	req.Args = []string{"machine", "backup"}
	return nil
}
```