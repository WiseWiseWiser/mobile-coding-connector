# Scenario

**Bug**: legacy server returned multipart ReadTimeout as HTTP 400; client must retry

```
# chunk 2 fails twice with 400 + "failed to parse form: ... i/o timeout", then succeeds
chunk[0,1] x1 -> chunk[2] x3 -> chunk[3,4] x1 -> complete
```

## Preconditions

Inherited from `transient-recovery/SETUP.md` size (5 x 2 MiB).

## Steps

1. Set `FlakyChunkIndex=2`, `TransientFails=2`, `FailStatus=400`.
2. Set `FailBody` to the crime-scene parse-form timeout message.

## Context

Proves timeout-as-400 is classified retryable (unlike a generic 400).

```go
import (
	"net/http"
	"testing"
	"github.com/xhd2015/doctest/session"
)

func Setup(t *testing.T, _ *session.Doctest, req *Request) error {
	req.TotalBytes = 10 * 1024 * 1024
	req.FlakyChunkIndex = 2
	req.TransientFails = 2
	req.FailStatus = http.StatusBadRequest
	req.FailBody = "failed to parse form: read tcp 127.0.0.1:23712->127.0.0.1:53826: i/o timeout"
	req.MaxChunkAttempts = 5
	req.AlwaysFailChunk = -1
	return nil
}
```
