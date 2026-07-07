package machinebackup

import (
	"fmt"
	"sort"
)

type excludedStats map[string]*excludedRuleStat

type excludedRuleStat struct {
	files int
	bytes int64
}

func newExcludedStats() excludedStats {
	return make(excludedStats)
}

func (s excludedStats) add(ruleKey string, files int, bytes int64) {
	if ruleKey == "" || files <= 0 {
		return
	}
	st, ok := s[ruleKey]
	if !ok {
		st = &excludedRuleStat{}
		s[ruleKey] = st
	}
	st.files += files
	st.bytes += bytes
}

func populateExcludedList(rules ExclusionRules, stats excludedStats) []ExcludePathEntry {
	out := make([]ExcludePathEntry, len(rules.ExcludedList))
	for i, e := range rules.ExcludedList {
		out[i] = e
		if st, ok := stats[e.Path]; ok {
			out[i].Files = st.files
			out[i].Bytes = st.bytes
		}
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].Bytes != out[j].Bytes {
			return out[i].Bytes > out[j].Bytes
		}
		return out[i].Path < out[j].Path
	})
	return out
}

func excludedTotals(entries []ExcludePathEntry) (paths, files int, bytes int64) {
	paths = len(entries)
	for _, e := range entries {
		files += e.Files
		bytes += e.Bytes
	}
	return paths, files, bytes
}

func formatExcludedSectionHeader(paths, files int, bytes int64) string {
	return fmt.Sprintf("  EXCLUDED (%d paths, %d files, %s)", paths, files, formatSize(bytes))
}

func formatExcludedColumnHeader() string {
	return fmt.Sprintf("    %-36s %8s %8s   %s", "RULE", "FILES", "SIZE", "REASON")
}

func formatExcludedRuleRow(ex ExcludePathEntry) string {
	return fmt.Sprintf("    %-36s %8d %8s   %s", ex.Path, ex.Files, formatSize(ex.Bytes), ex.Reason)
}