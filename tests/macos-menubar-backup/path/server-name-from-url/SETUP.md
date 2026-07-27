# Scenario

**Feature**: extract filesystem-safe server name from HTTPS URL

```
ServerNameFromURL("https://foo.example.com/") -> "foo.example.com"
```

## Preconditions

Active server URL includes scheme and trailing slash; host is the scope key.

## Steps

1. Set `Op=path_server_name`, URL `https://foo.example.com/`.

## Context

REQUIREMENT leaf: paths #8.

```go
import (
	"testing"

	"github.com/xhd2015/doctest/session"
)

func Setup(t *testing.T, _ *session.Doctest, req *Request) error {
	req.Op = "path_server_name"
	req.ServerURL = "https://foo.example.com/"
	return nil
}
```
