# Scenario

**Feature**: no usable token → empty token and source=none

```
# both sources missing or empty after trim
ResolveLocalServerToken -> token="", source=none
```

## Preconditions

Neither config nor credentials yields a non-empty trimmed token.

## Steps

1. Leaf omits config and/or seeds only blank credentials.
2. Expect empty token and `source=none` without fatal error.

## Context

REQUIREMENT group: `resolve/none/` (scenario 6).

```go
import (
	"testing"

	"github.com/xhd2015/doctest/session"
)

func Setup(t *testing.T, _ *session.Doctest, req *Request) error {
	req.ConfigPresent = false
	return nil
}
```
