# Scenario

**Feature**: paste-bin flag precedence over stdin detection

```
# --read overrides piped stdin write detection
piped junk + --read -> GET scratch (stdin ignored for PUT)
```

## Preconditions

`--read` must force read even when stdin is piped.

## Steps

1. Seed scratch with known content.
2. Pipe unrelated bytes and pass `--read`.
3. Assert read output matches seed and API content is unchanged.

## Context

Mode resolution: `--read` wins over piped-stdin write detection.

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