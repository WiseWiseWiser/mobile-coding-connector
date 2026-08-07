package synccmd

import (
	"regexp"
	"strconv"
	"strings"
)

// Sync outcomes for status display (no fake percentages).
const (
	OutcomeNever       = "never"
	OutcomeSynced      = "synced"      // last run: nothing to do
	OutcomePropagated  = "ok"          // last run: exit 0 with transfers
	OutcomeFailed      = "failed"
	OutcomePropagating = "propagated" // internal alias; display as "ok"
)

// parseUnisonOutcome derives outcome + counters from Unison text + exit code.
func parseUnisonOutcome(exitCode int, output string) (outcome, detail string, transferred, failed, skipped int) {
	low := strings.ToLower(output)
	transferred, failed, skipped = parseUnisonCounts(output)

	if exitCode != 0 || failed > 0 {
		outcome = OutcomeFailed
		if exitCode != 0 {
			detail = "exit " + strconv.Itoa(exitCode)
		}
		if failed > 0 {
			if detail != "" {
				detail += "; "
			}
			detail += strconv.Itoa(failed) + " failed"
		}
		if transferred > 0 {
			detail += ", " + strconv.Itoa(transferred) + " transferred"
		}
		if detail == "" {
			detail = "failed"
		}
		return outcome, detail, transferred, failed, skipped
	}

	// Success paths.
	if strings.Contains(low, "nothing to do") {
		return OutcomeSynced, "nothing to do", transferred, failed, skipped
	}
	if transferred > 0 {
		detail = strconv.Itoa(transferred) + " transferred"
		if skipped > 0 {
			detail += ", " + strconv.Itoa(skipped) + " skipped"
		}
		return OutcomePropagated, detail, transferred, failed, skipped
	}
	// Exit 0, no clear "nothing to do", zero transfers — treat as synced.
	if strings.Contains(low, "synchronization complete") || strings.Contains(low, "finished propagating") {
		return OutcomeSynced, "complete", transferred, failed, skipped
	}
	return OutcomeSynced, "ok", transferred, failed, skipped
}

var (
	reTransferred = regexp.MustCompile(`(?i)(\d+)\s+items?\s+transferred`)
	reFailed      = regexp.MustCompile(`(?i)(\d+)\s+failed`)
	reSkipped     = regexp.MustCompile(`(?i)(\d+)\s+skipped`)
)

func parseUnisonCounts(output string) (transferred, failed, skipped int) {
	// Prefer the summary line (often last non-empty).
	lines := strings.Split(output, "\n")
	for i := len(lines) - 1; i >= 0; i-- {
		line := strings.TrimSpace(lines[i])
		if line == "" {
			continue
		}
		low := strings.ToLower(line)
		if strings.Contains(low, "transferred") || strings.Contains(low, "synchronization") || strings.Contains(low, "nothing to do") {
			transferred = firstInt(reTransferred, line)
			failed = firstInt(reFailed, line)
			skipped = firstInt(reSkipped, line)
			return transferred, failed, skipped
		}
	}
	// Fallback whole buffer.
	transferred = firstInt(reTransferred, output)
	failed = firstInt(reFailed, output)
	skipped = firstInt(reSkipped, output)
	return transferred, failed, skipped
}

func firstInt(re *regexp.Regexp, s string) int {
	m := re.FindStringSubmatch(s)
	if len(m) < 2 {
		return 0
	}
	n, _ := strconv.Atoi(m[1])
	return n
}

// outcomeFromState maps stored state to display SYNC / LAST SYNC / DETAIL.
func outcomeFromState(st *pairState) (syncCol, lastSync, detail string) {
	if st == nil {
		return OutcomeNever, "—", "no successful run yet"
	}
	exit := 0
	if st.ExitCode != nil {
		exit = *st.ExitCode
	}
	outcome := strings.TrimSpace(st.Outcome)
	if outcome == "" {
		// Backward compat: derive from exit + message.
		outcome, detail, _, _, _ = parseUnisonOutcome(exit, st.Message)
	} else {
		detail = strings.TrimSpace(st.Message)
		if detail == "" {
			detail = outcome
		}
	}
	// Normalize internal "propagated" → display "ok".
	if outcome == OutcomePropagating || outcome == "propagated" {
		outcome = OutcomePropagated
	}
	if outcome == OutcomeFailed || exit != 0 {
		lastSync = "—"
		if st.LastRunAt != "" {
			// Still show attempt time in detail if no last success.
			if detail == "" || detail == outcome {
				detail = st.LastRunAt
			}
		}
		if outcome != OutcomeFailed {
			outcome = OutcomeFailed
		}
		return outcome, lastSync, detail
	}
	// Successful outcomes.
	lastSync = st.LastRunAt
	if lastSync == "" {
		lastSync = "—"
	}
	if outcome != OutcomeSynced && outcome != OutcomePropagated {
		if strings.Contains(strings.ToLower(st.Message), "nothing to do") {
			outcome = OutcomeSynced
		} else {
			outcome = OutcomePropagated
		}
	}
	if detail == "" || detail == outcome {
		if outcome == OutcomeSynced {
			detail = "nothing to do"
		} else if st.ItemsTransferred > 0 {
			detail = strconv.Itoa(st.ItemsTransferred) + " transferred"
		} else {
			detail = "ok"
		}
	}
	return outcome, lastSync, detail
}
