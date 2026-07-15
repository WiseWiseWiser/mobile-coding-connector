# Scenario

**Feature**: small write echoes content on stdout

```
# pipe hi -> paste-bin -> stderr saved 2 bytes + stdout hi
piped "hi" -> PUT scratch -> stderr summary + stdout echo
```

## Preconditions

Scratch reset before write.

## Steps

1. `resetScratch(req)`.
2. `setWritePipe(req, []byte(smallEchoPayload))`.

## Context

REQUIREMENT leaf: `write-small-echo`.

```go
import "testing"

func Setup(t *testing.T, req *Request) error {
	resetScratch(req)
	setWritePipe(t, req, []byte(smallEchoPayload))
	return nil
}
```