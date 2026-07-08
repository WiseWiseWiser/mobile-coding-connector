# Scenario

**Feature**: `remote-agent machine restore` applies tar.xz archives to server HOME

```
# prereq backup -> optional mutate serverHome -> restore --dry-run|apply via /restore/stream
CLASSIFYING: skip/update/create lines; APPLYING: (apply only) update/create; server summary log
```

## Preconditions

`serverHome` seeded; `Run` creates a prereq backup when `PrereqBackup` is true.

## Steps

1. Leaf sets `PrereqBackup`, `AfterBackupMutate`, and restore `Args`.
2. `Run` backs up, mutates server home if requested, then runs restore.
3. `Assert` checks CLASSIFYING/APPLYING sections, progress lines, summary title, and on-disk file contents.

## Context

Grouping node for restore: identical skips, changed dry-run plan, and apply.
Shared helpers parse restore SSE stdout: `CLASSIFYING:` (all classified entries),
optional `APPLYING:` (apply only), then `dry-run: machine restore plan` or
`machine restore summary` verbatim log block.

```go
import (
	"strings"
	"testing"
)

func Setup(t *testing.T, req *Request) error {
	if len(req.Args) >= 2 && req.Args[0] == "machine" && req.Args[1] != "restore" {
		t.Fatalf("restore group: unexpected subcommand argv %v", req.Args)
	}
	req.PrereqBackup = true
	return nil
}

// restoreClassifyingSection returns stdout from "CLASSIFYING:" through the last
// classify progress line (exclusive of APPLYING or summary).
func restoreClassifyingSection(combined string) string {
	idx := strings.Index(combined, "CLASSIFYING:\n")
	if idx < 0 {
		return ""
	}
	rest := combined[idx:]
	end := len(rest)
	for _, marker := range []string{"\nAPPLYING:\n", "\ndry-run: machine restore plan", "\nmachine restore summary"} {
		if i := strings.Index(rest, marker); i >= 0 && i < end {
			end = i
		}
	}
	return strings.TrimSuffix(rest[:end], "\n")
}

// restoreApplyingSection returns stdout from "APPLYING:" through the last apply
// progress line (exclusive of summary). Empty when APPLYING is absent.
func restoreApplyingSection(combined string) string {
	idx := strings.Index(combined, "APPLYING:\n")
	if idx < 0 {
		return ""
	}
	rest := combined[idx:]
	end := len(rest)
	for _, marker := range []string{"\ndry-run: machine restore plan", "\nmachine restore summary"} {
		if i := strings.Index(rest, marker); i >= 0 && i < end {
			end = i
		}
	}
	return strings.TrimSuffix(rest[:end], "\n")
}

// restoreSummaryRest returns the verbatim summary log block (dry-run or apply).
func restoreSummaryRest(combined string) string {
	for _, title := range []string{"dry-run: machine restore plan", "machine restore summary"} {
		if idx := strings.Index(combined, title); idx >= 0 {
			return combined[idx:]
		}
	}
	return ""
}

// assertRestoreStreamSections checks Option B section ordering: CLASSIFYING always;
// APPLYING only when wantApplying is true (real apply).
func assertRestoreStreamSections(t *testing.T, combined string, wantApplying bool) {
	t.Helper()
	classifyIdx := strings.Index(combined, "CLASSIFYING:\n")
	if classifyIdx < 0 {
		t.Fatalf("missing CLASSIFYING section; got:\n%s", combined)
	}
	classify := restoreClassifyingSection(combined)
	if classify == "" || classify == "CLASSIFYING:" {
		t.Fatalf("CLASSIFYING section has no progress lines; got:\n%s", combined)
	}

	hasApplying := strings.Contains(combined, "APPLYING:\n")
	if wantApplying {
		if !hasApplying {
			t.Fatalf("missing APPLYING section; got:\n%s", combined)
		}
		applyingIdx := strings.Index(combined, "APPLYING:\n")
		if classifyIdx > applyingIdx {
			t.Fatalf("CLASSIFYING must precede APPLYING; got:\n%s", combined)
		}
		applying := restoreApplyingSection(combined)
		if applying == "" || applying == "APPLYING:" {
			t.Fatalf("APPLYING section has no progress lines; got:\n%s", combined)
		}
	} else if hasApplying {
		t.Fatalf("unexpected APPLYING section in dry-run; got:\n%s", combined)
	}
}
```