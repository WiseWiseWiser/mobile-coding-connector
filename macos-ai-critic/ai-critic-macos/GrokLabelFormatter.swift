import Foundation

enum GrokLabelFormatter {
    private static let maxLabelLen = 40
    private static let prefix = "Grok "

    static func format(status: String, weeklyLimit: String, errorMsg: String) -> String {
        switch status {
        case "ready":
            return prefix + weeklyLimit
        case "loading":
            return prefix + "..."
        case "error":
            return truncate(prefix + errorMsg, max: maxLabelLen)
        default:
            return prefix + "..."
        }
    }

    private static func truncate(_ value: String, max: Int) -> String {
        let runes = Array(value)
        if runes.count <= max {
            return value
        }
        let ellipsis = Array("…")
        let keep = max - ellipsis.count
        if keep <= 0 {
            return String(runes.prefix(max))
        }
        return String(runes.prefix(keep)) + String(ellipsis)
    }
}