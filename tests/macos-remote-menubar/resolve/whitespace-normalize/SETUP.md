# Scenario

**Feature**: surrounding whitespace is trimmed for server match

```
default="  https://x.com/  " + domain https://x.com -> match, normalized base
```

## Preconditions

Normalize = trim space then trim trailing `/`.

## Steps

1. Set spaced default with trailing slash.

## Context

REQUIREMENT leaf: `resolve/whitespace-normalize` (extends normalize rules).

```go
import "testing"

func Setup(t *testing.T, req *Request) error {
	req.ConfigJSON = `{
  "default": "  https://x.com/  ",
  "domains": [
    {"server": "https://x.com", "token": "ws-tok"}
  ]
}`
	return nil
}
```
