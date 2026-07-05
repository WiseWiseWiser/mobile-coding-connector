import Foundation

enum UsageLabelFormatter {
    private static let maxLabelLen = 40

    static func formatGrokLabel(status: String, weeklyLimit: String, errorMsg: String) -> String {
        let prefix = "Grok "
        switch status {
        case "ready":
            return prefix + weeklyLimit
        case "loading":
            return prefix + "..."
        case "error":
            return prefix + "err"
        default:
            return prefix + "..."
        }
    }

    static func formatCodexLabel(status: String, monthlyUsage: String, errorMsg: String) -> String {
        let prefix = "Codex "
        switch status {
        case "ready":
            return prefix + monthlyUsage
        case "loading":
            return prefix + "..."
        case "error":
            return prefix + "err"
        default:
            return prefix + "..."
        }
    }

    static func formatMenuBarLabel(
        mode: String,
        rotatingIndex: Int,
        grokStatus: String,
        grokWeekly: String,
        grokError: String,
        codexStatus: String,
        codexMonthly: String,
        codexError: String
    ) -> String {
        switch mode {
        case "grok":
            return formatGrokLabel(status: grokStatus, weeklyLimit: grokWeekly, errorMsg: grokError)
        case "codex":
            return formatCodexLabel(status: codexStatus, monthlyUsage: codexMonthly, errorMsg: codexError)
        case "rotating":
            if rotatingIndex % 2 == 1 {
                return formatCodexLabel(status: codexStatus, monthlyUsage: codexMonthly, errorMsg: codexError)
            }
            return formatGrokLabel(status: grokStatus, weeklyLimit: grokWeekly, errorMsg: grokError)
        default:
            return formatGrokLabel(status: grokStatus, weeklyLimit: grokWeekly, errorMsg: grokError)
        }
    }

    static func formatGrokDropdownLine(status: String, weekly: String, reset: String, errorMsg: String) -> String {
        switch status {
        case "ready":
            return "Grok: Weekly Limit: \(weekly) (Reset \(reset))"
        case "loading":
            return "Grok: Loading..."
        case "error":
            return "Grok: Error: \(errorMsg)"
        default:
            return "Grok: Loading..."
        }
    }

    static func formatCodexDropdownLine(
        status: String,
        monthly: String,
        creditsUsed: String,
        creditsTotal: String,
        reset: String,
        errorMsg: String
    ) -> String {
        switch status {
        case "ready":
            return "Codex: Monthly Usage: \(monthly) — \(creditsUsed)/\(creditsTotal) (Reset \(reset))"
        case "loading":
            return "Codex: Loading..."
        case "error":
            return "Codex: Error: \(errorMsg)"
        default:
            return "Codex: Loading..."
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