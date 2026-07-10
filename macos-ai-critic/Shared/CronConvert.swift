import Foundation

/// Local ↔ UTC cron conversion for Cron Editor (mirrors macosapp/cronapi).
/// Safe simple 5-field expressions + fixed-offset zones only.
public enum CronConvert {
    public enum ConvertError: Error, LocalizedError {
        case invalidFields
        case unsafePattern(String)
        case variableOffset
        case wildcards
        case dayBoundary

        public var errorDescription: String? {
            switch self {
            case .invalidFields:
                return "Invalid cron expression (want 5 fields)"
            case .unsafePattern(let token):
                return "Unsafe cron pattern \(token): cannot auto-convert ranges/lists/steps"
            case .variableOffset:
                return "Timezone has DST or variable offset; cannot safely convert cron"
            case .wildcards:
                return "Unsafe cron: minute/hour wildcards need manual conversion"
            case .dayBoundary:
                return "Unsafe cron: conversion crosses day boundary with constrained date fields"
            }
        }
    }

    /// Convert local wall-time cron → UTC for API. Throws on unsafe patterns.
    public static func convertLocalCronToUTC(_ expr: String, timeZone: TimeZone = .current) throws -> String {
        let fields = expr.trimmingCharacters(in: .whitespacesAndNewlines)
            .split(whereSeparator: { $0.isWhitespace })
            .map(String.init)
        guard fields.count == 5 else { throw ConvertError.invalidFields }
        for f in fields {
            guard isSimpleCronToken(f) else { throw ConvertError.unsafePattern(f) }
        }
        guard zoneFixedOffset(timeZone) else { throw ConvertError.variableOffset }

        let minStr = fields[0]
        let hourStr = fields[1]
        if minStr == "*" || hourStr == "*" { throw ConvertError.wildcards }
        guard let min = Int(minStr), let hour = Int(hourStr) else {
            throw ConvertError.invalidFields
        }

        var cal = Calendar(identifier: .gregorian)
        cal.timeZone = timeZone
        let now = Date()
        var comps = cal.dateComponents([.year, .month, .day], from: now)
        comps.hour = hour
        comps.minute = min
        comps.second = 0
        guard let localDate = cal.date(from: comps) else { throw ConvertError.invalidFields }

        var utcCal = Calendar(identifier: .gregorian)
        utcCal.timeZone = TimeZone(secondsFromGMT: 0)!
        let utcComps = utcCal.dateComponents([.year, .month, .day, .hour, .minute], from: localDate)

        let localDay = cal.dateComponents([.year, .month, .day], from: localDate)
        if utcComps.day != localDay.day || utcComps.month != localDay.month || utcComps.year != localDay.year {
            if fields[2] != "*" || fields[3] != "*" || fields[4] != "*" {
                throw ConvertError.dayBoundary
            }
        }

        return "\(utcComps.minute ?? 0) \(utcComps.hour ?? 0) \(fields[2]) \(fields[3]) \(fields[4])"
    }

    /// Convert stored UTC cron → local for editor display. Throws on unsafe patterns.
    public static func convertUTCCronToLocal(_ expr: String, timeZone: TimeZone = .current) throws -> String {
        let fields = expr.trimmingCharacters(in: .whitespacesAndNewlines)
            .split(whereSeparator: { $0.isWhitespace })
            .map(String.init)
        guard fields.count == 5 else { throw ConvertError.invalidFields }
        for f in fields {
            guard isSimpleCronToken(f) else { throw ConvertError.unsafePattern(f) }
        }
        guard zoneFixedOffset(timeZone) else { throw ConvertError.variableOffset }

        let minStr = fields[0]
        let hourStr = fields[1]
        if minStr == "*" || hourStr == "*" { throw ConvertError.wildcards }
        guard let min = Int(minStr), let hour = Int(hourStr) else {
            throw ConvertError.invalidFields
        }

        var utcCal = Calendar(identifier: .gregorian)
        utcCal.timeZone = TimeZone(secondsFromGMT: 0)!
        let now = Date()
        var comps = utcCal.dateComponents([.year, .month, .day], from: now)
        comps.hour = hour
        comps.minute = min
        comps.second = 0
        guard let utcDate = utcCal.date(from: comps) else { throw ConvertError.invalidFields }

        var localCal = Calendar(identifier: .gregorian)
        localCal.timeZone = timeZone
        let localComps = localCal.dateComponents([.year, .month, .day, .hour, .minute], from: utcDate)

        let utcDay = utcCal.dateComponents([.year, .month, .day], from: utcDate)
        if localComps.day != utcDay.day || localComps.month != utcDay.month || localComps.year != utcDay.year {
            if fields[2] != "*" || fields[3] != "*" || fields[4] != "*" {
                throw ConvertError.dayBoundary
            }
        }

        return "\(localComps.minute ?? 0) \(localComps.hour ?? 0) \(fields[2]) \(fields[3]) \(fields[4])"
    }

    private static func isSimpleCronToken(_ f: String) -> Bool {
        if f == "*" { return true }
        if f.contains(where: { "-,/".contains($0) }) { return false }
        return Int(f) != nil
    }

    private static func zoneFixedOffset(_ tz: TimeZone) -> Bool {
        // Sample January and July offsets; equal → fixed (no DST).
        var cal = Calendar(identifier: .gregorian)
        cal.timeZone = TimeZone(secondsFromGMT: 0)!
        var jan = DateComponents()
        jan.year = 2024
        jan.month = 1
        jan.day = 15
        jan.hour = 12
        var jul = DateComponents()
        jul.year = 2024
        jul.month = 7
        jul.day = 15
        jul.hour = 12
        guard let janDate = cal.date(from: jan), let julDate = cal.date(from: jul) else {
            return false
        }
        return tz.secondsFromGMT(for: janDate) == tz.secondsFromGMT(for: julDate)
    }
}
