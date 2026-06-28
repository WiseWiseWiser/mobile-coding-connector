# Scenario

**Feature**: TTY user declines sing-box Homebrew install

```
# TTY + Confirm false: abort before brew
IsTTY -> Confirm("Install sing-box...") -> false -> error
```

## Steps

1. TTY with user declining confirm.

```go
import "testing"

func Setup(t *testing.T, req *Request) error {
	req.IsTTY = true
	no := false
	req.ConfirmYes = &no
	return nil
}
```