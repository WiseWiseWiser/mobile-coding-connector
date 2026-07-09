# Scenario

**Feature**: directory download rejected when remote source is missing

```
# browse/check fails before bytes transfer
absent remote path -> remote-agent download -> non-zero exit, no local mirror
```

## Preconditions

Remote source path does not exist under `serverHome` before download.

## Steps

1. Leaf sets `download` args for a missing remote path.
2. Assertions expect non-zero exit, actionable error text, and no local files created.

## Context

Download must fail before writing partial local tree.

```go
import (
	"testing"

	"github.com/xhd2015/ai-critic/script/lib"
)

func Setup(t *testing.T, req *Request) error {
	if req.Token == "" {
		req.Token = lib.TestPassword
	}
	return nil
}
```