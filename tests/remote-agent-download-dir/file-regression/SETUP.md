# Scenario

**Feature**: single-file download regression

```
# existing GET download path for a regular file
remote file -> remote-agent download -> one local file, no directory markers
```

## Preconditions

Leaf seeds a remote file fixture on `serverHome`; no local pre-seed required unless noted.

## Steps

1. Leaf seeds `serverHome` and calls `setDownloadArgs`.
2. Assertions expect exit 0, success stdout, and matching bytes locally.

## Context

Guards that directory downloads do not break the existing file-download code path.

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