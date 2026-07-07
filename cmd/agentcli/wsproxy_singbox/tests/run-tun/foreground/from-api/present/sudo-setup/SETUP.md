# Scenario

**Feature**: ws-proxy vpn auto sudoers setup via sudosetup before sing-box launch

```
# needSudo + default: EnsureSudoSetup(sing-box path) before RunSingBox
# --no-setup-sudo: skip EnsureSudoSetup, rely on sudo timestamp cache only
```

## Preconditions

- sing-box is on PATH (`SingBoxOnPath=true`).
- Non-root EUID with TTY (foreground sudo path).
- `EnsureSudoSetup` hook replaces real sudosetup.Manager (no real sudo).

## Steps

1. Inherit `present` Setup (`SingBoxOnPath=true`).
2. Leaf narrows `NoSetupSudo` and mock install state.

```go
import "testing"

func Setup(t *testing.T, req *Request) error {
	req.IsTTY = true
	if req.EUID == nil {
		req.EUID = euidPtr(1000)
	}
	return nil
}
```