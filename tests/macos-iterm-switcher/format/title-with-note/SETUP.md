# Scenario

**Feature**: title with note

```
title-with-note -> formatter / client contract
```

```go
import (
	"testing"
	"github.com/xhd2015/doctest/session"
)
func Setup(t *testing.T, _ *session.Doctest, req *Request) error {
	req.Op = "format"
	req.Name = "grok review"
	req.Note = "fix auth on staging"
	req.Cwd = "~/proj"
	req.SessionID = "sess-a"
	return nil
}
```
