# Scenario

**Feature**: more than 10 backups today keeps newest 10

```
12 files on 2026-07-10 -> keep 10 newest, delete 2 oldest today
```

## Preconditions

All entries share today's local calendar day relative to now.

## Steps

1. Twelve entries hours 00..11 on 2026-07-10; now=15:00Z.

## Context

REQUIREMENT #17.

```go
import (
	"encoding/json"
	"fmt"
	"testing"
)

func Setup(t *testing.T, req *Request) error {
	req.NowRFC3339 = "2026-07-10T15:00:00Z"
	type e struct {
		Path      string `json:"path"`
		ModTime   string `json:"mod_time"`
		SizeBytes int64  `json:"size_bytes"`
	}
	var list []e
	for h := 0; h < 12; h++ {
		list = append(list, e{
			Path:      fmt.Sprintf("today-%02d.tar.xz", h),
			ModTime:   fmt.Sprintf("2026-07-10T%02d:00:00Z", h),
			SizeBytes: int64(h + 1),
		})
	}
	b, err := json.Marshal(list)
	if err != nil {
		return err
	}
	req.EntriesJSON = string(b)
	return nil
}
```
