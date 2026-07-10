import Foundation

/// Pure backup menu labels / schedule / retention helpers.
/// Mirrors `macosapp/menubar` backup helpers used by the remote menubar app.
public enum BackupMenuFormatter {
    public static let backupIntervalSeconds = 3600

    public enum Phase: String {
        case off
        case idle
        case running
        case error
    }

    public struct TaskStatus {
        public var enabled: Bool
        public var phase: Phase
        public var lastFinishedAt: Date?
        public var nextRunAt: Date?
        public var lastError: String
        public var lastSizeBytes: Int64

        public init(
            enabled: Bool = false,
            phase: Phase = .off,
            lastFinishedAt: Date? = nil,
            nextRunAt: Date? = nil,
            lastError: String = "",
            lastSizeBytes: Int64 = 0
        ) {
            self.enabled = enabled
            self.phase = phase
            self.lastFinishedAt = lastFinishedAt
            self.nextRunAt = nextRunAt
            self.lastError = lastError
            self.lastSizeBytes = lastSizeBytes
        }
    }

    public struct FileEntry: Identifiable, Equatable {
        public var id: String { path }
        public var path: String
        public var modTime: Date
        public var sizeBytes: Int64

        public init(path: String, modTime: Date, sizeBytes: Int64) {
            self.path = path
            self.modTime = modTime
            self.sizeBytes = sizeBytes
        }
    }

    /// Host-only server scope key (no scheme/path/slash).
    public static func serverNameFromURL(_ serverURL: String) -> String {
        let trimmed = serverURL.trimmingCharacters(in: .whitespacesAndNewlines)
        guard !trimmed.isEmpty else { return "" }
        var toParse = trimmed
        if !trimmed.contains("://") {
            toParse = "https://" + trimmed
        }
        if let url = URL(string: toParse), let host = url.host, !host.isEmpty {
            return host
        }
        var s = trimmed
        if s.hasPrefix("https://") { s = String(s.dropFirst("https://".count)) }
        if s.hasPrefix("http://") { s = String(s.dropFirst("http://".count)) }
        if let idx = s.firstIndex(where: { $0 == "/" || $0 == "?" || $0 == "#" }) {
            s = String(s[..<idx])
        }
        return s.trimmingCharacters(in: CharacterSet(charactersIn: "/"))
    }

    public static func backupDir(home: String, serverName: String) -> String {
        let step1 = (home as NSString).appendingPathComponent(".backup")
        let step2 = (step1 as NSString).appendingPathComponent("ai-critic")
        return (step2 as NSString).appendingPathComponent(serverName)
    }

    /// Resolve backup dir under the user's home for a server URL.
    public static func backupDirForServerURL(_ serverURL: String, home: String = NSHomeDirectory()) -> String {
        backupDir(home: home, serverName: serverNameFromURL(serverURL))
    }

    public static func backupArchiveFilename(utc: Date) -> String {
        var calendar = Calendar(identifier: .gregorian)
        calendar.timeZone = TimeZone(secondsFromGMT: 0)!
        let c = calendar.dateComponents([.year, .month, .day, .hour, .minute, .second], from: utc)
        let y = c.year ?? 0
        let mo = c.month ?? 0
        let d = c.day ?? 0
        let h = c.hour ?? 0
        let mi = c.minute ?? 0
        let s = c.second ?? 0
        return String(format: "machine-backup-%04d%02d%02d-%02d%02d%02d.tar.xz", y, mo, d, h, mi, s)
    }

    public static func shouldRunOnEnable(lastFinished: Date?, now: Date, interval: TimeInterval) -> Bool {
        guard let last = lastFinished else { return true }
        return now.timeIntervalSince(last) > interval
    }

    public static func shouldRunDue(enabled: Bool, running: Bool, nextRunAt: Date?, now: Date) -> Bool {
        if !enabled || running { return false }
        guard let next = nextRunAt else { return true }
        return next.timeIntervalSince(now) <= 0
    }

    public static func formatBackupStatusTitle(_ st: TaskStatus, now: Date = Date()) -> String {
        if !st.enabled || st.phase == .off {
            return "Status: Off"
        }
        switch st.phase {
        case .running:
            return "Status: On · Running"
        case .error:
            let rel = formatRelPast(st.lastFinishedAt, now: now)
            return "Status: On · Error · \(rel)"
        case .idle, .off:
            let last = formatRelPast(st.lastFinishedAt, now: now)
            let next = formatRelFuture(st.nextRunAt, now: now)
            return "Status: On · last \(last) · next \(next)"
        }
    }

    public static func formatBackupEntry(_ entry: FileEntry, now: Date = Date()) -> String {
        "\(formatRelPast(entry.modTime, now: now)) · \(formatHumanSize(entry.sizeBytes))"
    }

    public static func formatBackupRecentEmptyLabel() -> String {
        "No recent backups"
    }

    public static func sortBackupEntriesNewestFirst(_ entries: [FileEntry]) -> [FileEntry] {
        entries.sorted { $0.modTime > $1.modTime }
    }

    public static func pruneBackupFiles(_ entries: [FileEntry], now: Date = Date()) -> (keep: [FileEntry], delete: [FileEntry]) {
        let keepTodayN = 10
        let dailyDays = 7
        let cal = Calendar.current
        let todayStart = cal.startOfDay(for: now)
        guard let windowStart = cal.date(byAdding: .day, value: -dailyDays, to: todayStart) else {
            return (entries, [])
        }

        var byDay: [Date: [FileEntry]] = [:]
        for e in entries {
            let day = cal.startOfDay(for: e.modTime)
            byDay[day, default: []].append(e)
        }

        var keep: [FileEntry] = []
        var delete: [FileEntry] = []
        for (day, list) in byDay {
            let sorted = list.sorted { $0.modTime > $1.modTime }
            if day == todayStart {
                for (i, e) in sorted.enumerated() {
                    if i < keepTodayN { keep.append(e) } else { delete.append(e) }
                }
            } else if day >= windowStart && day < todayStart {
                for (i, e) in sorted.enumerated() {
                    if i == 0 { keep.append(e) } else { delete.append(e) }
                }
            } else {
                delete.append(contentsOf: sorted)
            }
        }
        return (keep, delete)
    }

    public static func backupStatusMenuChildren() -> [String] {
        ["Enable", "Disable"]
    }

    public static func backupEnableItemEnabled(_ enabled: Bool) -> Bool { !enabled }
    public static func backupDisableItemEnabled(_ enabled: Bool) -> Bool { enabled }

    // MARK: - Relative / size

    private static func formatRelPast(_ date: Date?, now: Date) -> String {
        guard let date else { return "0m ago" }
        var d = now.timeIntervalSince(date)
        if d < 0 { d = 0 }
        return "\(formatRelDuration(d)) ago"
    }

    private static func formatRelFuture(_ date: Date?, now: Date) -> String {
        guard let date else { return "in 0m" }
        var d = date.timeIntervalSince(now)
        if d < 0 { d = 0 }
        return "in \(formatRelDuration(d))"
    }

    private static func formatRelDuration(_ d: TimeInterval) -> String {
        let totalMinutes = Int(d / 60)
        let totalHours = Int(d / 3600)
        let totalDays = Int(d / 86400)
        if totalDays >= 1 { return "\(totalDays)d" }
        if totalHours >= 1 { return "\(totalHours)h" }
        if totalMinutes < 1 {
            return d == 0 ? "0m" : "1m"
        }
        return "\(totalMinutes)m"
    }

    private static func formatHumanSize(_ n: Int64) -> String {
        let kb: Int64 = 1024
        let mb: Int64 = 1024 * 1024
        let gb: Int64 = 1024 * 1024 * 1024
        var v = n
        if v < 0 { v = 0 }
        if v >= gb { return "\(v / gb) GB" }
        if v >= mb { return "\(v / mb) MB" }
        if v >= kb { return "\(v / kb) KB" }
        return "\(v) B"
    }
}
