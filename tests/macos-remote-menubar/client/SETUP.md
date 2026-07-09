# Scenario

**Feature**: remote product source contracts (menu + install identity)

```
Swift remote product / shared profile-gated menu -> no Restart Daemon for remote
script/macos-app/install-remote.sh -> app name + bundle id, no server binary embed
```

## Preconditions

Read-only inspection of module sources. Leaves are RED until implementer adds
`install-remote.sh` and remote product / profile gating.

## Steps

1. Set `Op=client`.
2. Leaf sets `ClientLeaf`.

## Context

REQUIREMENT group: `client/`.

```go
import "testing"

func Setup(t *testing.T, req *Request) error {
	req.Op = "client"
	return nil
}
```
