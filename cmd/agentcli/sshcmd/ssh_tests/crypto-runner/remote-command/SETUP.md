# Scenario

**Feature**: CryptoSSHRunner.Run remote command against AdhocServer

```
# command mode
runner.Run(sess, ["echo","hello"], opts) -> stdout contains hello
```

## Preconditions

- Scenario: `crypto-runner-command`.
- RemoteArgv `["echo","hello"]`; EchoNeedle `hello`.

## Steps

1. Adhoc + key pair; Session.LocalPort = Adhoc port.
2. CryptoSSHRunner with Signer + InsecureIgnoreHostKey.
3. Run; Assert stdout needle.

## Context

- Implements P1 SSHRunner interface.

```go
import (
	"testing"

	"github.com/xhd2015/doctest/session"
)

func Setup(t *testing.T, d *session.Doctest, req *Request) error {
	t.Helper()
	_ = d
	req.Scenario = ScenarioCryptoRunnerCommand
	req.RemoteCommand = "echo hello"
	req.RemoteArgv = []string{"echo", "hello"}
	req.EchoNeedle = "hello"
	return nil
}
```
