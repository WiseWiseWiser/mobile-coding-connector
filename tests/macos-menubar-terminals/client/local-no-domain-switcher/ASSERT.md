## Expected

1. `LocalHasDomainSwitcher` is `false`.

## Side Effects

- None (read-only source inspection).

## Errors

- Local app gained a Server/domain switcher meant for remote multi-domain.

```go
import (
	"testing"

	"github.com/xhd2015/doctest/session"
)

func Assert(t *testing.T, _ *session.Doctest, req *Request, resp *Response, err error) {
	if err != nil {
		t.Fatal(err)
	}
	if resp.LocalHasDomainSwitcher {
		t.Fatalf("local app must not have remote domain/Server switcher (sources: %v)", resp.SwiftSourcesChecked)
	}
}
```
