# Scenario

**Feature**: run history older than 7 days is pruned

```
# seed task with recentRuns: one old (>8d) + one recent
server load/tick/list/history -> old run absent; recent kept
```

## Preconditions

1. Seed `recentRuns` inline on the task definition (allowed layout choice).
2. Old run startedAt ~10 days before now (fixed RFC3339 far in the past).
3. Recent run startedAt near "now" (use a timestamp within last day).

## Steps

1. Seed disabled task so scheduler does not add noise (or enabled with long interval).
2. Action `history` (or list) to trigger read/prune path.
3. Assert history does not contain the old startedAt; may contain the recent one.

## Context

Priority leaf optional prune. If implementer stores history elsewhere, seed file must
still be honored at boot or prune-on-read must drop old rows when history is returned.

```go
import "testing"

func Setup(t *testing.T, req *Request) error {
	req.SeedTasks = []TaskSeed{
		{
			ID:           "prune-hist",
			Name:         "prune-hist",
			Command:      "echo prune",
			ScheduleMode: "interval",
			Interval:     "1h",
			Timeout:      "1h",
			Enabled:      boolPtr(false),
			RecentRuns: []map[string]any{
				{
					"startedAt":  "2020-01-01T00:00:00Z",
					"finishedAt": "2020-01-01T00:00:01Z",
					"exitCode":   0,
				},
				{
					"startedAt":  "2026-07-09T12:00:00Z",
					"finishedAt": "2026-07-09T12:00:01Z",
					"exitCode":   0,
				},
			},
		},
	}
	req.Target = "prune-hist"
	req.Action = "history"
	// small wait for any boot prune tick
	req.WaitSecs = 2
	return nil
}
```
