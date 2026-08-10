## Expected

1. No error.
2. `DefaultPort == 23891`.
3. `Config.Port == 23891`.
4. `Config.Disabled == false`.
5. `Config.Token` empty.

## Errors

- Wrong default port constant or resolve mapping.

```go
import (
	"testing"

	"github.com/xhd2015/doctest/session"
)

func Assert(t *testing.T, d *session.Doctest, req *Request, resp *Response, err error) {
	t.Helper()
	if err != nil {
		t.Fatalf("Run error: %v", err)
	}
	const want = 23891
	if resp.DefaultPort != want {
		t.Fatalf("DefaultPublishPort() = %d, want %d", resp.DefaultPort, want)
	}
	if resp.Config.Port != want {
		t.Fatalf("Config.Port = %d, want %d", resp.Config.Port, want)
	}
	if resp.Config.Disabled {
		t.Fatal("Config.Disabled true, want false")
	}
	if resp.Config.Token != "" {
		t.Fatalf("Config.Token = %q, want empty", resp.Config.Token)
	}
}
```
