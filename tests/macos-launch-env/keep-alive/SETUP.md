# Scenario

**Feature**: keep-alive child must not auto-open browser

```
binaryDir -> KeepAliveEnv -> AI_CRITIC_NO_OPEN_BROWSER=1
```

## Preconditions

`KeepAliveEnv` is shared by all keep-alive env leaves.

## Steps

1. Leaf setup may override `BinaryDir`.
2. Assert checks browser-suppression env only (no usage-bin paths).

## Context

Grouping for REQUIREMENT-DESIGN-in-process-usage-fetch.md Part B `tests/macos-launch-env/`.

```go
import (
	"strings"
	"testing"
)

func Setup(t *testing.T, req *Request) error {
	if req.BinaryDir == "" {
		req.BinaryDir = "/app/Contents/MacOS"
	}
	req.BinaryDir = strings.TrimSuffix(req.BinaryDir, "/")
	return nil
}
```