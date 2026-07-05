# Scenario

**Bug**: install must not open browser when keep-alive starts from menu-bar app

```
KeepAliveEnv(dir) -> env contains AI_CRITIC_NO_OPEN_BROWSER=1
```

## Preconditions

Server opens browser only when `AI_CRITIC_NO_OPEN_BROWSER` is unset.

## Steps

1. Use default bundle MacOS dir from root setup.

## Context

REQUIREMENT leaf: `keep-alive/no-open-browser`.

```go
import "testing"

func Setup(t *testing.T, req *Request) error {
	req.BinaryDir = "/app/Contents/MacOS"
	return nil
}
```