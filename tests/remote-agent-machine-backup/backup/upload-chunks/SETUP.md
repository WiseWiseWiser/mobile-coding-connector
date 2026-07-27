# Scenario

**Feature**: backup dry-run excludes **/upload-chunks segment trees

```
# walk applies upload-chunks segment rule -> EXCLUDED lists rule; chunk files omitted
stream: DOT DIRS omits upload-chunks content
```

## Preconditions

`serverHome` includes `.live-and-love/upload-chunks/chunk-1`.

## Steps

1. Args: `machine backup --dry-run`.

## Context

REQUIREMENT leaf `backup/upload-chunks`.

```go
import (
	"testing"

	"github.com/xhd2015/doctest/session"
)

func Setup(t *testing.T, _ *session.Doctest, req *Request) error {
	req.Args = []string{"machine", "backup", "--dry-run"}
	return nil
}
```