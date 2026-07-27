# Scenario

**Feature**: default with trailing slash matches domain without slash

```
default=https://x.com/ + domain https://x.com -> match; Server normalized without trailing slash
```

## Preconditions

Normalize trims trailing `/` for match and for returned base URL.

## Steps

1. Set default with trailing slash; domain without.

## Context

REQUIREMENT leaf: `resolve/trailing-slash-match`.

```go
import (
	"testing"

	"github.com/xhd2015/doctest/session"
)

func Setup(t *testing.T, _ *session.Doctest, req *Request) error {
	req.ConfigJSON = `{
  "default": "https://x.com/",
  "domains": [
    {"server": "https://x.com", "token": "tok-x"}
  ]
}`
	return nil
}
```
