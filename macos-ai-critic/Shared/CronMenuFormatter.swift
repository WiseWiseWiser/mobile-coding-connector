import Foundation

/// Cron submenu labels/actions — mirrors `macosapp/menubar` cron formatters.
public enum CronMenuFormatter {
    /// `{name} {glyph status} · {sched}` e.g. `backup ● Running · every 5m`.
    public static func formatCronTaskTitle(
        name: String,
        status: String,
        enabled: Bool,
        scheduleMode: String,
        interval: String,
        cronExpr: String
    ) -> String {
        let statusPart = formatStatusPart(status: status, enabled: enabled)
        let sched = formatSchedule(scheduleMode: scheduleMode, interval: interval, cronExpr: cronExpr)
        return "\(name) \(statusPart) · \(sched)"
    }

    private static func formatStatusPart(status: String, enabled: Bool) -> String {
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
            let label = status.isEmpty ? "Idle" : status
            if !enabled {
                return "○ \(label) (disabled)"
            }
            return "○ \(label)"
        }
    }

    private static func formatSchedule(scheduleMode: String, interval: String, cronExpr: String) -> String {
        switch scheduleMode {
        case "interval":
            return "every \(interval)"
        case "cron":
            return "cron \(cronExpr)"
        default:
            if !interval.isEmpty {
                return "every \(interval)"
            }
            if !cronExpr.isEmpty {
                return "cron \(cronExpr)"
            }
            return scheduleMode
        }
    }

    /// Run Now enabled unless status is running.
    public static func canRunCronTask(status: String) -> Bool {
        status != "running"
    }

    /// Delete… enabled unless status is running (mirrors CanDeleteCronTask).
    public static func canDeleteCronTask(status: String) -> Bool {
        status != "running"
    }

    /// Confirm dialog copy before DELETE.
    public static func formatDeleteCronConfirm(name: String) -> String {
        "Delete cron task \"\(name)\"?"
    }

    /// true → show Enable; false → show Disable.
    public static func showEnableCronAction(enabled: Bool) -> Bool {
        !enabled
    }

    public static func formatCronTasksEmptyLabel() -> String {
        "No cron tasks configured"
    }

    public static func formatCronNotConfiguredLabel() -> String {
        "Not configured"
    }

    /// Prefer non-empty trimmed server message; else `Task updated`.
    public static func cronToggleAlertMessage(serverMessage: String) -> String {
        let trimmed = serverMessage.trimmingCharacters(in: .whitespacesAndNewlines)
        if !trimmed.isEmpty {
            return trimmed
        }
        return "Task updated"
    }
}
