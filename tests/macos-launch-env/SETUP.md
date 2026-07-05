# Scenario

**Feature**: keep-alive launch environment for macOS menu-bar app

```
binaryDir -> KeepAliveEnv -> child env map for ai-critic keep-alive
```

## Preconditions

1. `macosapp/launchenv` exports `KeepAliveEnv(binaryDir string) map[string]string`.
2. No subprocess — pure env construction.

## Steps

1. Root `Setup` sets default `BinaryDir` when leaf omits it.
2. Leaf `Assert` checks required env keys and values.

## Context

Implements REQUIREMENT-DESIGN-macos-app-install-fixes.md Bug 1 env extraction and
Bug 2 bundled usage-bin paths for DaemonManager.

```go
import "testing"

func Setup(t *testing.T, req *Request) error {
	if req.BinaryDir == "" {
		req.BinaryDir = "/app/Contents/MacOS"
	}
	return nil
}
```