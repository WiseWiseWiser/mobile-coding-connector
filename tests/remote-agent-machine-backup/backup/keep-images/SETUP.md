# Scenario

**Feature**: backup keeps user images under .live-and-love/imgs/

```
# JPEG magic bytes -> included in DOT FILES and archive (not **(binary))
```

## Preconditions

`serverHome` includes `.live-and-love/imgs/photo.jpg` with JPEG SOI bytes.

## Steps

1. Set `OutputPath` under `agentHome`.
2. Args: `machine backup --output <path>`.

## Context

REQUIREMENT leaf `backup/keep-images`.

```go
import "testing"

func Setup(t *testing.T, req *Request) error {
	req.OutputPath = "keep-images-backup.tar.xz"
	req.Args = []string{"machine", "backup", "--output", "__OUTPUT_PATH__"}
	return nil
}
```