# Scenario

**Feature**: resume continues partial file via HTTP Range

```
# local big.bin has first 512 B; remote has 1024 B -> resumed at 50% -> full file
pre-seeded half file -> remote-agent download -> resumed at + complete bytes
```

## Preconditions

Remote `uploads/mirror/big.bin` is 1024 bytes; local `local-mirror/big.bin` pre-seeded with first 512 bytes.

## Steps

1. Seed remote `big.bin` with deterministic 1024-byte pattern.
2. Pre-seed local `big.bin` with first 512 bytes of same pattern.
3. Args: `download uploads/mirror ./local-mirror`.

## Context

REQUIREMENT leaf #5 — dir-success/resume-partial-file.

```go
import "testing"

const partialFileSize = 1024
const partialPrefill = 512

func Setup(t *testing.T, req *Request) error {
	full := string(repeatBytePattern(partialFileSize, 42))
	req.ServerPreseedFiles = map[string]string{
		"uploads/mirror/big.bin": full,
	}
	setDownloadArgs(t, req, "uploads/mirror", "./local-mirror")
	req.LocalDir = localDirRel("uploads/mirror", "./local-mirror")
	req.LocalPreseedFiles = map[string]string{
		"big.bin": full[:partialPrefill],
	}
	return nil
}
```