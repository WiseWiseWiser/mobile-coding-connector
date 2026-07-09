# Scenario

**Feature**: single-file upload regression

```
# existing chunked upload path for a regular file
local file -> remote-agent upload -> one remote file, unchanged CLI semantics
```

## Preconditions

Leaf creates a local file fixture; no server pre-seed required unless noted.

## Steps

1. Leaf copies or writes a local file and calls `setUploadArgs`.
2. Assertions expect exit 0, success stdout, and matching bytes on the server.

## Context

Guards that directory uploads do not break the existing file-upload code path.

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