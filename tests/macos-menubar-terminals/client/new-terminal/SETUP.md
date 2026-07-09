# Scenario

**Feature**: New Terminal action present in both apps

```
Terminals menu / menu body -> "New Terminal" (no prompt) on local and remote
```

## Preconditions

Both products expose New Terminal… (or New Terminal) without cwd prompt for v1.

## Steps

1. Set `ClientLeaf=new-terminal`.

## Context

REQUIREMENT: New Terminal present on both apps.

```go
import "testing"

func Setup(t *testing.T, req *Request) error {
	req.ClientLeaf = "new-terminal"
	return nil
}
```
