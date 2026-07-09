# Scenario

**Feature**: Open in Browser uses resolved remote server URL

```
ResolvedEndpoint -> remoteconfig.OpenBrowserURL -> remote base URL (not 127.0.0.1 keep-alive)
```

## Preconditions

Remote app opens the configured remote server, not the local keep-alive port.

## Steps

1. Set `Op=browser`.
2. Leaf sets resolved endpoint fields.

## Context

REQUIREMENT group: `browser/`.

```go
import "testing"

func Setup(t *testing.T, req *Request) error {
	req.Op = "browser"
	return nil
}
```
