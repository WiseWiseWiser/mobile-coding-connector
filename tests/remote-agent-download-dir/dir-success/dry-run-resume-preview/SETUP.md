# Scenario

**Feature**: download dry-run previews skip and resume without mutating local bytes

```
# complete a.txt + sub/b.txt, partial big.bin locally -> --dry-run -> would skip/resume, bytes unchanged
pre-seeded local mirror -> remote-agent download --dry-run -> resume preview stdout only
```

## Preconditions

Remote tree at `uploads/mirror` with `a.txt`, `sub/b.txt`, and 1024-byte `big.bin`.
Local `local-mirror/` has complete copies of `a.txt` and `sub/b.txt` plus first 512 bytes of `big.bin`.

## Steps

1. Seed standard remote tree plus `big.bin` on server.
2. Pre-seed local complete files and partial `big.bin`.
3. Args: `download --dry-run uploads/mirror ./local-mirror`.

## Context

REQUIREMENT-DESIGN-upload-download-dry-run.md — dir-success/dry-run-resume-preview.

```go
import "testing"

const resumePreviewFileSize = 1024
const resumePreviewPrefill = 512

func Setup(t *testing.T, req *Request) error {
	full := string(repeatBytePattern(resumePreviewFileSize, 42))
	seedStandardRemoteTree(req)
	req.ServerPreseedFiles["uploads/mirror/big.bin"] = full
	setDownloadDryRunArgs(t, req, "uploads/mirror", "./local-mirror")
	req.LocalDir = localDirRel("uploads/mirror", "./local-mirror")
	req.LocalPreseedFiles = map[string]string{
		"a.txt":     "alpha\n",
		"sub/b.txt": "bravo\n",
		"big.bin":   full[:resumePreviewPrefill],
	}
	return nil
}
```