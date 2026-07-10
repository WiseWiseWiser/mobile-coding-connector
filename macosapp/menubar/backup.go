package menubar

import (
	"fmt"
	"net/url"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

// BackupIntervalSeconds is the default periodic machine-backup interval (1 hour).
const BackupIntervalSeconds = 3600

// BackupPhase is the display/runtime phase of a periodic backup task.
type BackupPhase string

const (
	BackupPhaseOff     BackupPhase = "off"
	BackupPhaseIdle    BackupPhase = "idle"
	BackupPhaseRunning BackupPhase = "running"
	BackupPhaseError   BackupPhase = "error"
)

// BackupTaskStatus is the pure status model used for menu titles and schedule policy.
type BackupTaskStatus struct {
	Enabled        bool
	Phase          BackupPhase
	LastFinishedAt time.Time
	NextRunAt      time.Time
	LastError      string
	LastSizeBytes  int64
}

// BackupFileEntry describes one local archive file for recent list / retention.
type BackupFileEntry struct {
	Path      string
	ModTime   time.Time
	SizeBytes int64
}

// ServerNameFromURL returns a filesystem-safe server scope key (host only).
func ServerNameFromURL(serverURL string) string {
	s := strings.TrimSpace(serverURL)
	if s == "" {
		return ""
	}
	// url.Parse requires a scheme for Host; add a synthetic one when missing.
	toParse := s
	if !strings.Contains(s, "://") {
		toParse = "https://" + s
	}
	u, err := url.Parse(toParse)
	if err != nil {
		// Fallback: strip scheme and path slashes manually.
		s = strings.TrimPrefix(s, "https://")
		s = strings.TrimPrefix(s, "http://")
		if i := strings.IndexAny(s, "/?#"); i >= 0 {
			s = s[:i]
		}
		return strings.TrimSuffix(s, "/")
	}
	host := u.Hostname()
	if host == "" {
		host = strings.Trim(u.Host, "/")
	}
	return host
}

// BackupDir returns {home}/.backup/ai-critic/{serverName}.
func BackupDir(home, serverName string) string {
	return filepath.Join(home, ".backup", "ai-critic", serverName)
}

// BackupArchiveFilename returns machine-backup-<YYYYMMDD-HHMMSS>.tar.xz in UTC.
func BackupArchiveFilename(utc time.Time) string {
	return "machine-backup-" + utc.UTC().Format("20060102-150405") + ".tar.xz"
}

// ShouldRunOnEnable reports whether enabling the task should kick off a run now.
// never ran → true; last finish ≤ interval ago → false; last finish > interval → true.
func ShouldRunOnEnable(lastFinished, now time.Time, interval time.Duration) bool {
	if lastFinished.IsZero() {
		return true
	}
	return now.Sub(lastFinished) > interval
}

// ShouldRunDue reports whether a scheduled run should start.
// disabled / already running → false; enabled, idle, nextRunAt <= now → true.
func ShouldRunDue(enabled, running bool, nextRunAt, now time.Time) bool {
	if !enabled || running {
		return false
	}
	if nextRunAt.IsZero() {
		return true
	}
	return !nextRunAt.After(now)
}

// FormatBackupStatusTitle renders the nested Status menu title.
func FormatBackupStatusTitle(st BackupTaskStatus, now time.Time) string {
	if !st.Enabled || st.Phase == BackupPhaseOff {
		return "Status: Off"
	}
	switch st.Phase {
	case BackupPhaseRunning:
		return "Status: On · Running"
	case BackupPhaseError:
		rel := formatBackupRelPast(st.LastFinishedAt, now)
		return "Status: On · Error · " + rel
	default:
		// idle (and unknown-on)
		last := formatBackupRelPast(st.LastFinishedAt, now)
		next := formatBackupRelFuture(st.NextRunAt, now)
		return fmt.Sprintf("Status: On · last %s · next %s", last, next)
	}
}

// FormatBackupEntry formats a recent-list row: "{rel past} · {human size}".
func FormatBackupEntry(entry BackupFileEntry, now time.Time) string {
	rel := formatBackupRelPast(entry.ModTime, now)
	size := formatBackupHumanSize(entry.SizeBytes)
	return rel + " · " + size
}

// FormatBackupRecentEmptyLabel is shown when the backup directory has no archives.
func FormatBackupRecentEmptyLabel() string {
	return "No recent backups"
}

// SortBackupEntriesNewestFirst returns a copy sorted by ModTime descending.
func SortBackupEntriesNewestFirst(entries []BackupFileEntry) []BackupFileEntry {
	out := append([]BackupFileEntry(nil), entries...)
	sort.SliceStable(out, func(i, j int) bool {
		return out[i].ModTime.After(out[j].ModTime)
	})
	return out
}

// PruneBackupFiles applies retention:
//   - today (local calendar of now): keep newest 10
//   - past days within 7 calendar days [today-7, today-1]: keep 1 newest per day
//   - older: delete all
func PruneBackupFiles(entries []BackupFileEntry, now time.Time) (keep, delete []BackupFileEntry) {
	const keepTodayN = 10
	const dailyDays = 7

	loc := now.Location()
	nowLocal := now.In(loc)
	todayY, todayM, todayD := nowLocal.Date()
	today := time.Date(todayY, todayM, todayD, 0, 0, 0, 0, loc)
	// Past window: inclusive of today-7 through yesterday.
	windowStart := today.AddDate(0, 0, -dailyDays)

	type dayKey struct {
		y int
		m time.Month
		d int
	}
	dayOf := func(t time.Time) dayKey {
		y, m, d := t.In(loc).Date()
		return dayKey{y, m, d}
	}

	// Group by calendar day.
	byDay := map[dayKey][]BackupFileEntry{}
	for _, e := range entries {
		k := dayOf(e.ModTime)
		byDay[k] = append(byDay[k], e)
	}

	for k, dayEntries := range byDay {
		// Newest first within the day.
		sort.SliceStable(dayEntries, func(i, j int) bool {
			return dayEntries[i].ModTime.After(dayEntries[j].ModTime)
		})
		dayStart := time.Date(k.y, k.m, k.d, 0, 0, 0, 0, loc)

		switch {
		case dayStart.Equal(today):
			for i, e := range dayEntries {
				if i < keepTodayN {
					keep = append(keep, e)
				} else {
					delete = append(delete, e)
				}
			}
		case !dayStart.Before(windowStart) && dayStart.Before(today):
			// Within past 7 calendar days: keep newest only.
			for i, e := range dayEntries {
				if i == 0 {
					keep = append(keep, e)
				} else {
					delete = append(delete, e)
				}
			}
		default:
			// Older / outside window.
			delete = append(delete, dayEntries...)
		}
	}
	return keep, delete
}

// BackupStatusMenuChildren returns the only children of the nested Status menu.
func BackupStatusMenuChildren() []string {
	return []string{"Enable", "Disable"}
}

// BackupEnableItemEnabled is true when the task is currently off.
func BackupEnableItemEnabled(enabled bool) bool {
	return !enabled
}

// BackupDisableItemEnabled is true when the task is currently on.
func BackupDisableItemEnabled(enabled bool) bool {
	return enabled
}

// formatBackupRelPast returns "Nm ago" / "Nh ago" / "Nd ago".
func formatBackupRelPast(t, now time.Time) string {
	if t.IsZero() {
		return "0m ago"
	}
	d := now.Sub(t)
	if d < 0 {
		d = 0
	}
	return formatBackupRelDuration(d) + " ago"
}

// formatBackupRelFuture returns "in Nm" / "in Nh" / "in Nd".
func formatBackupRelFuture(t, now time.Time) string {
	if t.IsZero() {
		return "in 0m"
	}
	d := t.Sub(now)
	if d < 0 {
		d = 0
	}
	return "in " + formatBackupRelDuration(d)
}

func formatBackupRelDuration(d time.Duration) string {
	if d < 0 {
		d = 0
	}
	totalMinutes := int(d / time.Minute)
	totalHours := int(d / time.Hour)
	totalDays := int(d / (24 * time.Hour))
	if totalDays >= 1 {
		return fmt.Sprintf("%dd", totalDays)
	}
	if totalHours >= 1 {
		return fmt.Sprintf("%dh", totalHours)
	}
	if totalMinutes < 1 {
		// Sub-minute: still show 0m for exact equality; sealed leaves use exact minutes.
		if d == 0 {
			return "0m"
		}
		return "1m"
	}
	return fmt.Sprintf("%dm", totalMinutes)
}

// formatBackupHumanSize uses binary units (1024) with whole numbers for exact multiples.
// Labels are KB/MB/GB (not KiB/MiB).
func formatBackupHumanSize(n int64) string {
	if n < 0 {
		n = 0
	}
	const (
		kb = 1024
		mb = 1024 * 1024
		gb = 1024 * 1024 * 1024
	)
	switch {
	case n >= gb && n%gb == 0:
		return fmt.Sprintf("%d GB", n/gb)
	case n >= gb:
		return fmt.Sprintf("%d GB", n/gb)
	case n >= mb && n%mb == 0:
		return fmt.Sprintf("%d MB", n/mb)
	case n >= mb:
		return fmt.Sprintf("%d MB", n/mb)
	case n >= kb && n%kb == 0:
		return fmt.Sprintf("%d KB", n/kb)
	case n >= kb:
		return fmt.Sprintf("%d KB", n/kb)
	default:
		return fmt.Sprintf("%d B", n)
	}
}

// CanRunBackupNow reports whether Backup Now can start.
// Independent of periodic task enabled: one-shot is always allowed when ready.
func CanRunBackupNow(hasEndpoint, running bool, serverName string) bool {
	if !hasEndpoint || running {
		return false
	}
	return strings.TrimSpace(serverName) != ""
}

// ShouldShowBackupProgressWindow is true for manual / enable-immediate runs;
// false for hourly schedule ticks.
func ShouldShowBackupProgressWindow(triggeredBySchedule bool) bool {
	return !triggeredBySchedule
}

// FormatBackupProgressStartHeader returns "Machine backup — {server}".
func FormatBackupProgressStartHeader(serverName string) string {
	return "Machine backup — " + serverName
}

// FormatBackupProgressStartedAt returns "Started 2006-01-02 15:04:05" using t as-is (no Local convert).
func FormatBackupProgressStartedAt(t time.Time) string {
	return "Started " + t.Format("2006-01-02 15:04:05")
}

// FormatBackupProgressWindowTitle returns "Backup: {server}" or "Backup: (no server)".
func FormatBackupProgressWindowTitle(serverName string) string {
	if strings.TrimSpace(serverName) == "" {
		return "Backup: (no server)"
	}
	return "Backup: " + serverName
}

// FormatBackupProgressSection returns "[section] {message}".
func FormatBackupProgressSection(message string) string {
	return "[section] " + message
}

// FormatBackupProgressFrame returns "[progress] {name} {status}" and optional " — {detail}".
func FormatBackupProgressFrame(name, status, detail string) string {
	line := "[progress] " + name + " " + status
	if detail != "" {
		line += " — " + detail
	}
	return line
}

// FormatBackupProgressLog returns the message verbatim (no [log] prefix).
func FormatBackupProgressLog(message string) string {
	return message
}

// FormatBackupProgressError returns "ERROR: {message}".
func FormatBackupProgressError(message string) string {
	return "ERROR: " + message
}

// FormatBackupProgressDone returns "[done] archive ready" when message is empty.
func FormatBackupProgressDone(message string) string {
	if strings.TrimSpace(message) == "" {
		return "[done] archive ready"
	}
	return "[done] " + message
}

// FormatBackupProgressDownloadStart returns "Downloading archive…" (U+2026).
func FormatBackupProgressDownloadStart() string {
	return "Downloading archive…"
}

// FormatBackupProgressWrote returns "Wrote {path} ({human size})".
func FormatBackupProgressWrote(path string, sizeBytes int64) string {
	return fmt.Sprintf("Wrote %s (%s)", path, formatBackupHumanSize(sizeBytes))
}

// FormatBackupProgressStatusSuccess returns "Status: Success".
func FormatBackupProgressStatusSuccess() string {
	return "Status: Success"
}

// FormatBackupProgressStatusFailed returns "Status: Failed".
func FormatBackupProgressStatusFailed() string {
	return "Status: Failed"
}

// FormatBackupProgressGuardError maps guard reasons to ERROR lines.
// "not_configured" → "ERROR: not configured"; "no_server" → "ERROR: no server selected".
func FormatBackupProgressGuardError(reason string) string {
	switch reason {
	case "not_configured":
		return "ERROR: not configured"
	case "no_server":
		return "ERROR: no server selected"
	default:
		return "ERROR: " + reason
	}
}
