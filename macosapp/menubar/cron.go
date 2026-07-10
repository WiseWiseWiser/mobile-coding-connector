package menubar

import (
	"fmt"
	"strings"
)

// FormatCronTaskTitle renders the per-cron-task submenu title:
// `{name} {glyph status} · {sched}` e.g. `backup ● Running · every 5m`.
func FormatCronTaskTitle(name, status string, enabled bool, scheduleMode, interval, cronExpr string) string {
	return fmt.Sprintf("%s %s · %s", name, formatCronStatusPart(status, enabled), formatCronSchedule(scheduleMode, interval, cronExpr))
}

func formatCronStatusPart(status string, enabled bool) string {
	switch status {
	case "running":
		return "● Running"
	case "error":
		return "⚠ Error"
	case "idle":
		if !enabled {
			return "○ Idle (disabled)"
		}
		return "○ Idle"
	default:
		label := status
		if label == "" {
			label = "Idle"
		}
		if !enabled {
			return fmt.Sprintf("○ %s (disabled)", label)
		}
		return fmt.Sprintf("○ %s", label)
	}
}

func formatCronSchedule(scheduleMode, interval, cronExpr string) string {
	switch scheduleMode {
	case "interval":
		return "every " + interval
	case "cron":
		return "cron " + cronExpr
	default:
		if interval != "" {
			return "every " + interval
		}
		if cronExpr != "" {
			return "cron " + cronExpr
		}
		return scheduleMode
	}
}

// CanRunCronTask reports whether Run Now should be enabled.
// False only when status is "running".
func CanRunCronTask(status string) bool {
	return status != "running"
}

// CanDeleteCronTask reports whether Delete… should be enabled.
// False only when status is "running".
func CanDeleteCronTask(status string) bool {
	return status != "running"
}

// FormatDeleteCronConfirm is the NSAlert confirm copy before DELETE.
func FormatDeleteCronConfirm(name string) string {
	return fmt.Sprintf(`Delete cron task "%s"?`, name)
}

// ShowEnableCronAction reports whether the menu should offer Enable instead of Disable.
func ShowEnableCronAction(enabled bool) bool {
	return !enabled
}

// FormatCronTasksEmptyLabel is shown when the cron task list is empty.
func FormatCronTasksEmptyLabel() string {
	return "No cron tasks configured"
}

// FormatCronNotConfiguredLabel is shown when the remote endpoint is missing.
func FormatCronNotConfiguredLabel() string {
	return "Not configured"
}

// CronToggleAlertMessage returns NSAlert copy for enable/disable.
// Non-empty trimmed server message wins; otherwise "Task updated".
func CronToggleAlertMessage(serverMessage string) string {
	if msg := strings.TrimSpace(serverMessage); msg != "" {
		return msg
	}
	return "Task updated"
}
