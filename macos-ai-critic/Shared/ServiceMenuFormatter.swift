import Foundation

/// Service submenu labels/actions — mirrors `macosapp/menubar` service formatters.
public enum ServiceMenuFormatter {
    public static func formatServiceTitle(name: String, status: String, enabled: Bool) -> String {
        switch status {
        case "running":
            return "\(name) ● Running"
        case "error":
            return "\(truncateName(name, maxRunes: 1)) ⚠ Error"
        case "stopped":
            if !enabled {
                return "\(name) ○ Stopped (disabled)"
            }
            return "\(name) ○ Stopped"
        case "starting":
            return "\(name) ○ Starting"
        default:
            if !enabled {
                return "\(name) ○ Stopped (disabled)"
            }
            return "\(name) ○ \(status)"
        }
    }

    public static func canStopService(pid: Int, desiredRunning: Bool) -> Bool {
        if pid > 0 { return true }
        return desiredRunning
    }

    public static func showEnableAction(enabled: Bool) -> Bool {
        !enabled
    }

    public static func formatServicesEmptyLabel() -> String {
        "No services configured"
    }

    private static func truncateName(_ name: String, maxRunes: Int) -> String {
        let runes = Array(name)
        if runes.count <= maxRunes {
            return name
        }
        let keep = max(0, maxRunes - 1)
        return String(runes.prefix(keep)) + "…"
    }
}
