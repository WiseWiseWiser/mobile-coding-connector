# Scenario

**Feature**: registry tracks launch and stop clears port

```
launch grok -> opencode-serve-children.json -> stop -> port closed, registry empty
```

## Preconditions

- Fake opencode when real not in PATH; real opencode allowed with `slow` label.

## Steps

1. `Op = OpRegistryLaunchStop`.

## Context

Integration leaf for Path A registry + shared cleanup helper.

```go
import "testing"

func Setup(t *testing.T, req *Request) error {
	req.Op = OpRegistryLaunchStop
	return nil
}
```
