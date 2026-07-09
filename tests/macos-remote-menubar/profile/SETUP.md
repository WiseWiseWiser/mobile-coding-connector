# Scenario

**Feature**: local vs remote app profile flags

```
appprofile.Local() | appprofile.Remote() -> SpawnsDaemon, config file, display/bundle identity
```

## Preconditions

`macosapp/appprofile` exports `Local()` and `Remote()` with the product table
from REQUIREMENT-DESIGN-remote-agent-macos-bar-app.md.

## Steps

1. Set `Op=profile`.
2. Leaf sets `ProfileName` to `local` or `remote`.

## Context

REQUIREMENT group: `profile/`.

```go
import "testing"

func Setup(t *testing.T, req *Request) error {
	req.Op = "profile"
	return nil
}
```
