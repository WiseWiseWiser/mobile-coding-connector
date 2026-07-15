# Scenario

**Feature**: CollectOpencodeServePIDs discovers children from registry and ports

```
opencode-serve-children.json -> Collect -> PID list
lsof tcp:PORT (fake opencode) -> Collect -> PID list
```

## Preconditions

- Registry fixture or live fake opencode listener per leaf.

## Steps

1. Set `Op = OpCollect`.

## Context

MECE: file-based discovery vs port-based discovery.

```go
import "testing"

func Setup(t *testing.T, req *Request) error {
	req.Op = OpCollect
	return nil
}
```
