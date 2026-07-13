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

    /// Compose-only dropdown from structured API fields (no parse of next_reset).
    static func composeGrokDropdownLine(
        status: String,
        weekly: String,
        resetDisplay: String,
        timeLeft: String,
        errorMsg: String
    ) -> String {
        switch status {
        case "ready":
            var line = "Grok: \(weekly)(Weekly), Reset \(resetDisplay)"
            if !timeLeft.isEmpty {
                line += ", \(timeLeft)"
            }
            return line
        case "loading":
            return "Grok: Loading..."
        case "error":
            return "Grok: Error: \(errorMsg)"
        default:
            return "Grok: Loading..."
        }
    }

    /// Compose-only dropdown from structured API fields (no parse of next_reset).
    static func composeCodexDropdownLine(
        status: String,
        monthly: String,
        creditsUsed: String,
        creditsTotal: String,
        resetDisplay: String,
        timeLeft: String,
        errorMsg: String
    ) -> String {
        switch status {
        case "ready":
            var line = "Codex: \(monthly)(Monthly) \(creditsUsed)/\(creditsTotal), Reset \(resetDisplay)"
            if !timeLeft.isEmpty {
                line += ", \(timeLeft)"
            }
            return line
        case "loading":
            return "Codex: Loading..."
        case "error":
            return "Codex: Error: \(errorMsg)"
        default:
            return "Codex: Loading..."
        }
    }

    /// Legacy path that re-parses raw next_reset. Prefer compose* with API A+B fields.
    static func formatGrokDropdownLine(status: String, weekly: String, reset: String, errorMsg: String, now: Date = Date()) -> String {
        switch status {
        case "ready":
            return composeGrokDropdownLine(
                status: status,
                weekly: weekly,
                resetDisplay: formatResetDisplay(reset: reset, now: now),
                timeLeft: formatTimeLeft(reset: reset, now: now),
                errorMsg: errorMsg
            )
        default:
            return composeGrokDropdownLine(status: status, weekly: weekly, resetDisplay: "", timeLeft: "", errorMsg: errorMsg)
        }
    }

    /// Legacy path that re-parses raw next_reset. Prefer compose* with API A+B fields.
    static func formatCodexDropdownLine(
        status: String,
        monthly: String,
        creditsUsed: String,
        creditsTotal: String,
        reset: String,
        errorMsg: String,
        now: Date = Date()
    ) -> String {
        switch status {
        case "ready":
            return composeCodexDropdownLine(
                status: status,
                monthly: monthly,
                creditsUsed: creditsUsed,
                creditsTotal: creditsTotal,
                resetDisplay: formatResetDisplay(reset: reset, now: now),
                timeLeft: formatTimeLeft(reset: reset, now: now),
                errorMsg: errorMsg
            )
        default:
            return composeCodexDropdownLine(
                status: status,
                monthly: monthly,
                creditsUsed: creditsUsed,
                creditsTotal: creditsTotal,
                resetDisplay: "",
                timeLeft: "",
                errorMsg: errorMsg
            )
        }
    }

    private static let grokResetRegex = try! NSRegularExpression(
        pattern: #"^(\w+)\s+(\d+),\s+(\d+):(\d+)(?::(\d+))?\s+PT$"#
    )
    private static let codexResetRegex = try! NSRegularExpression(
        pattern: #"^(\d+):(\d+)\s+on\s+(\d+)\s+(\w+)$"#
    )
    private static let monthByName: [String: Int] = [
        "january": 1, "jan": 1,
        "february": 2, "feb": 2,
        "march": 3, "mar": 3,
        "april": 4, "apr": 4,
        "may": 5,
        "june": 6, "jun": 6,
        "july": 7, "jul": 7,
        "august": 8, "aug": 8,
        "september": 9, "sep": 9, "sept": 9,
        "october": 10, "oct": 10,
        "november": 11, "nov": 11,
        "december": 12, "dec": 12,
    ]
    private static let pacificTimeZone = TimeZone(identifier: "America/Los_Angeles") ?? TimeZone(secondsFromGMT: -8 * 3600)!

    static func formatResetDisplay(reset: String, now: Date) -> String {
        let trimmed = reset.trimmingCharacters(in: .whitespacesAndNewlines)
        if trimmed.isEmpty {
            return trimmed
        }

        let isGrok = grokResetRegex.firstMatch(in: trimmed, range: NSRange(trimmed.startIndex..., in: trimmed)) != nil
        let isCodex = codexResetRegex.firstMatch(in: trimmed, range: NSRange(trimmed.startIndex..., in: trimmed)) != nil
        guard let resetTime = parseResetTime(reset: reset, now: now) else {
            return reset
        }

        var calendar = Calendar(identifier: .gregorian)
        calendar.timeZone = TimeZone.current
        let components = calendar.dateComponents([.year, .month, .day, .hour, .minute], from: resetTime)

        if isGrok {
            guard let month = components.month, let day = components.day,
                  let hour = components.hour, let minute = components.minute else {
                return reset
            }
            let monthName = grokMonthName(month)
            return String(format: "%@ %d, %02d:%02d", monthName, day, hour, minute)
        }
        if isCodex {
            guard let month = components.month, let day = components.day,
                  let hour = components.hour, let minute = components.minute else {
                return reset
            }
            let monthAbbrev = codexMonthAbbrev(month)
            return String(format: "%@ %d, %02d:%02d", monthAbbrev, day, hour, minute)
        }
        return reset
    }

    static func formatTimeLeft(reset: String, now: Date) -> String {
        guard let resetTime = parseResetTime(reset: reset, now: now) else {
            return ""
        }
        let remaining = resetTime.timeIntervalSince(now)
        if remaining <= 0 {
            return "left 0m"
        }
        let totalHours = Int(remaining / 3600)
        if totalHours >= 24 {
            let days = Int(remaining / (24 * 3600))
            let hours = totalHours % 24
            if hours == 0 {
                return "left \(days)d"
            }
            return "left \(days)d\(hours)h"
        }
        if totalHours >= 1 {
            let minutes = Int(remaining / 60) % 60
            if minutes == 0 {
                return "left \(totalHours)h"
            }
            return "left \(totalHours)h\(minutes)m"
        }
        var mins = Int(remaining / 60)
        if mins < 1 {
            mins = 1
        }
        return "left \(mins)m"
    }

    private static func parseResetTime(reset: String, now: Date) -> Date? {
        let trimmed = reset.trimmingCharacters(in: .whitespacesAndNewlines)
        if trimmed.isEmpty {
            return nil
        }

        if let match = grokResetRegex.firstMatch(in: trimmed, range: NSRange(trimmed.startIndex..., in: trimmed)),
           match.numberOfRanges >= 5,
           let monthName = captureGroup(match, in: trimmed, index: 1),
           let dayText = captureGroup(match, in: trimmed, index: 2),
           let hourText = captureGroup(match, in: trimmed, index: 3),
           let minuteText = captureGroup(match, in: trimmed, index: 4),
           let month = monthNumber(from: monthName),
           let day = Int(dayText),
           let hour = Int(hourText),
           let minute = Int(minuteText) {
            let secondText = captureGroup(match, in: trimmed, index: 5)
            let second = secondText.flatMap(Int.init) ?? 0
            var calendar = Calendar(identifier: .gregorian)
            calendar.timeZone = pacificTimeZone
            let year = calendar.component(.year, from: now)
            var components = DateComponents(
                year: year,
                month: month,
                day: day,
                hour: hour,
                minute: minute,
                second: second
            )
            guard var resetTime = calendar.date(from: components) else {
                return nil
            }
            if resetTime < now {
                components.year = year + 1
                resetTime = calendar.date(from: components) ?? resetTime
            }
            return resetTime
        }

        if let match = codexResetRegex.firstMatch(in: trimmed, range: NSRange(trimmed.startIndex..., in: trimmed)),
           match.numberOfRanges >= 5,
           let hourText = captureGroup(match, in: trimmed, index: 1),
           let minuteText = captureGroup(match, in: trimmed, index: 2),
           let dayText = captureGroup(match, in: trimmed, index: 3),
           let monthName = captureGroup(match, in: trimmed, index: 4),
           let hour = Int(hourText),
           let minute = Int(minuteText),
           let day = Int(dayText),
           let month = monthNumber(from: monthName) {
            var calendar = Calendar(identifier: .gregorian)
            calendar.timeZone = TimeZone.current
            let year = calendar.component(.year, from: now)
            var components = DateComponents(
                year: year,
                month: month,
                day: day,
                hour: hour,
                minute: minute,
                second: 0
            )
            guard var resetTime = calendar.date(from: components) else {
                return nil
            }
            if resetTime < now {
                components.year = year + 1
                resetTime = calendar.date(from: components) ?? resetTime
            }
            return resetTime
        }

        return nil
    }

    private static func monthNumber(from name: String) -> Int? {
        monthByName[name.lowercased()]
    }

    private static let grokMonthNames = [
        "January", "February", "March", "April", "May", "June",
        "July", "August", "September", "October", "November", "December",
    ]
    private static let codexMonthAbbrevs = [
        "Jan", "Feb", "Mar", "Apr", "May", "Jun",
        "Jul", "Aug", "Sep", "Oct", "Nov", "Dec",
    ]

    private static func grokMonthName(_ month: Int) -> String {
        guard month >= 1, month <= 12 else { return "" }
        return grokMonthNames[month - 1]
    }

    private static func codexMonthAbbrev(_ month: Int) -> String {
        guard month >= 1, month <= 12 else { return "" }
        return codexMonthAbbrevs[month - 1]
    }

    private static func captureGroup(_ match: NSTextCheckingResult, in text: String, index: Int) -> String? {
        let range = match.range(at: index)
        guard range.location != NSNotFound, let swiftRange = Range(range, in: text) else {
            return nil
        }
        return String(text[swiftRange])
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