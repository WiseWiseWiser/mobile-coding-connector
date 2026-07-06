# Scenario

**Feature**: repeatable --exclude merges with built-in exclusions

```
# seed .docker/config, backup with --exclude .docker
.docker omitted from plan and archive
```

## Preconditions

`serverHome` includes `.docker/config` (via `SeedDocker`).

## Steps

1. `SeedDocker=true`, `ExcludePaths=[".docker"]`.
2. Dry-run then stream backup to `custom-exclude.tar.xz`.

## Context

REQUIREMENT leaf `backup/custom-exclude`.

```go
import "testing"

func Setup(t *testing.T, req *Request) error {
	req.SeedDocker = true
	req.ExcludePaths = []string{".docker"}
	req.OutputPath = "custom-exclude.tar.xz"
	req.Args = []string{"machine", "backup", "--output", "__OUTPUT_PATH__"}
	return nil
}
```