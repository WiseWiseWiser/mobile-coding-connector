# Scenario

**Feature**: default path gzip-compresses when smaller

```
compressible file -> init compressed=true -> complete size = original
```

```go
import (
	"bytes"
	"testing"
	"github.com/xhd2015/doctest/session"
)

func Setup(t *testing.T, _ *session.Doctest, req *Request) error {
	req.NoCompress = false
	req.Content = bytes.Repeat([]byte("compress-me-please-"), 64*1024)
	return nil
}
```
