# Scenario

**Feature**: ShellQuote handles spaces and apostrophes without injection

```
# special chars
shell.ShellQuote("arg with spaces") -> safe token, no injection with adjacent text
shell.ShellQuote("it's") -> safe token, round-trip preserved
```

```go
import (
	"testing"
	"github.com/xhd2015/doctest/session"
)
func Setup(t *testing.T, _ *session.Doctest, req *Request) error {
	req.Phase = "shell-quote-special"
	req.ShellQuoteInputs = []string{"arg with spaces", "it's"}
	return nil
}
```