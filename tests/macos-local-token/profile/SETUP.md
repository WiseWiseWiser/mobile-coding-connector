# Scenario

**Feature**: local app profile advertises auth-token usage

```
appprofile.Local() -> UsesAuthToken=true (local ServerClient requires Bearer)
```

## Preconditions

Once local menu-bar attaches Bearer tokens, the local profile flag should reflect
that capability (parallel to remote profile).

## Steps

1. Set `Op=profile`.
2. Leaf sets `ProfileName`.

## Context

REQUIREMENT group: `profile/` (optional scenario 9 / UsesAuthToken).

```go
import "testing"

func Setup(t *testing.T, req *Request) error {
	req.Op = "profile"
	return nil
}
```
