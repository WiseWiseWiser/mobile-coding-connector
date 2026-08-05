# Scenario

**Feature**: CryptoSSHRunner — x/crypto SSH client implementing SSHRunner

```
# runner dials session LocalPort (here = Adhoc port, no relay)
CryptoSSHRunner{Signer,Stdout}.Run(sess, remoteArgv)
  -> ssh.Dial 127.0.0.1:sess.LocalPort
  -> session.Run(join argv) | Shell when argv empty
```

## Preconditions

- Scenario family: crypto-runner.
- CryptoSSHRunner may be missing until implementer (RED).

## Steps

1. Start Adhoc with authorized client key.
2. Build Session{LocalPort: adhoc port, User, Alive}.
3. CryptoSSHRunner.Run with remote argv; capture Stdout.

## Context

- Isolates runner from ServeService/LocalRelay (those covered under through-relay).

```go
import (
	"testing"

	"github.com/xhd2015/doctest/session"
)

func Setup(t *testing.T, d *session.Doctest, req *Request) error {
	t.Helper()
	_ = d
	_ = req
	return nil
}
```
