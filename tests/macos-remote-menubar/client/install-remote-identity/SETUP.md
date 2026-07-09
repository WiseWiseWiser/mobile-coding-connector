# Scenario

**Feature**: install-remote identity matches product table

```
script/macos-app/install-remote.sh -> APP_NAME=ai-critic-remote-macos,
  BUNDLE_ID=com.xhd2015.ai-critic-remote-macos, does not embed server binary
```

## Preconditions

`install-remote.sh` exists (or will be added by implementer).

## Steps

1. Set `ClientLeaf=install-remote-identity`.

## Context

REQUIREMENT leaf: `client/install-remote-identity`. RED until script exists.

```go
import "testing"

func Setup(t *testing.T, req *Request) error {
	req.ClientLeaf = "install-remote-identity"
	return nil
}
```
