# Scenario

**Feature**: simple alphanumeric path produces shell-safe embeddable token

```
# simple path quote
shell.ShellQuote("/tmp/ai-critic") -> sh -c round-trip -> "/tmp/ai-critic"
```

```go
import "testing"

func Setup(t *testing.T, req *Request) error {
	req.Phase = "shell-quote-simple"
	req.ShellQuoteInput = "/tmp/ai-critic"
	return nil
}
```