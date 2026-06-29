# Scenario

**Feature**: TTY user accepts Homebrew install for sing-box

```
# TTY + Confirm true -> BrewInstall -> RunSingBox
Confirm yes -> BrewInstall("sing-box") -> RunSingBox
```

## Steps

1. TTY with user accepting confirm; non-root needs sudo after install.

```go
import "testing"

func Setup(t *testing.T, req *Request) error {
	req.IsTTY = true
	yes := true
	req.ConfirmYes = &yes
	req.EUID = euidPtr(1000)
	return nil
}
```