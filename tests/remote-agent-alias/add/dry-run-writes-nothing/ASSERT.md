## Expected

1. Exit 0.
2. Stdout has `[dry-run] would add alias xdev → <xdev URL>` and
   `[dry-run] would write ~/.local/bin/xdev-agent`.
3. No wrapper file and no alias store were created.

## Exit Code

0.

```go
import (
	"os"
	"strings"
	"testing"

	"github.com/xhd2015/doctest/session"
)

func Assert(t *testing.T, _ *session.Doctest, req *Request, resp *Response, err error) {
	if err != nil {
		t.Fatalf("Run error: %v", err)
	}
	if resp.ExitCode != 0 {
		t.Fatalf("exit %d; combined:\n%s", resp.ExitCode, resp.Combined)
	}
	if !strings.Contains(resp.Stdout, "[dry-run] would add alias "+aliasLeafName+" → "+resp.AliasURL) {
		t.Fatalf("stdout missing the add plan:\n%s", resp.Stdout)
	}
	if !strings.Contains(resp.Stdout, "[dry-run] would write ~/.local/bin/"+aliasLeafBinary) {
		t.Fatalf("stdout missing the write plan:\n%s", resp.Stdout)
	}
	if resp.WrapperAfter != "" {
		t.Fatalf("dry-run wrote a wrapper: %q", resp.WrapperAfter)
	}
	if _, statErr := os.Stat(resp.Wrapper); !os.IsNotExist(statErr) {
		t.Fatalf("wrapper file exists after dry-run (stat err = %v)", statErr)
	}
	if resp.StoreJSON != "" {
		t.Fatalf("dry-run wrote the alias store: %s", resp.StoreJSON)
	}
}
```
