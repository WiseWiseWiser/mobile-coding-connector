# Scenario

**Feature**: Kill terminates fake opencode serve child

```
fake opencode serve -> Kill(registry pid) -> port closed, process gone
```

## Preconditions

- Fake opencode on ephemeral port with matching registry entry.

## Steps

1. `Op = OpKill`, `StartFakeOpenCode = true`, `UseRegistryPID = true`.

## Context

Happy-path kill for headless agent port cleanup.

```go
import "testing"

func Setup(t *testing.T, req *Request) error {
	req.Op = OpKill
	req.StartFakeOpenCode = true
	req.UseRegistryPID = true
	return nil
}
```
