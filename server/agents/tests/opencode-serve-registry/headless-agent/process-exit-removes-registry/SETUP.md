# Scenario

**Feature**: external child exit removes registry entry

```
launch -> SIGKILL child pid -> monitor removes registry entry
```

## Preconditions

- Registry records child PID; process monitor watches `cmd.Wait()`.

## Steps

1. `Op = OpExitRegistry`, `KillChildExternally = true`.

## Context

Orphan prevention when child dies without explicit stop API.

```go
import "testing"

func Setup(t *testing.T, req *Request) error {
	req.Op = OpExitRegistry
	req.KillChildExternally = true
	req.UseFakeOpenCode = true
	return nil
}
```
