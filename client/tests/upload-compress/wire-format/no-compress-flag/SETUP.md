# Scenario

**Feature**: `--no-compress` / `NoCompress` uploads raw bytes

```
same compressible file + NoCompress -> compressed=false; wire size = original
```

```go
import (
	"bytes"
	"testing"
	"github.com/xhd2015/doctest/session"
)

func Setup(t *testing.T, _ *session.Doctest, req *Request) error {
	req.NoCompress = true
	req.Content = bytes.Repeat([]byte("compress-me-please-"), 64*1024)
	return nil
}
```
