import Foundation

/// Per-server periodic backup task state under `~/.ai-critic/periodic-backup.json`.
/// Default: enabled = false (task off until user enables).
public struct PeriodicBackupServerState: Codable, Equatable {
    public var enabled: Bool
    public var intervalSeconds: Int
    public var lastStartedAt: String
    public var lastFinishedAt: String
    public var lastStatus: String
    public var lastError: String
    public var lastOutputPath: String
    public var lastSizeBytes: Int64
    public var nextRunAt: String

    public init(
        enabled: Bool = false,
        intervalSeconds: Int = BackupMenuFormatter.backupIntervalSeconds,
        lastStartedAt: String = "",
        lastFinishedAt: String = "",
        lastStatus: String = "idle",
        lastError: String = "",
        lastOutputPath: String = "",
        lastSizeBytes: Int64 = 0,
        nextRunAt: String = ""
    ) {
        self.enabled = enabled
        self.intervalSeconds = intervalSeconds
        self.lastStartedAt = lastStartedAt
        self.lastFinishedAt = lastFinishedAt
        self.lastStatus = lastStatus
        self.lastError = lastError
        self.lastOutputPath = lastOutputPath
        self.lastSizeBytes = lastSizeBytes
        self.nextRunAt = nextRunAt
    }

    enum CodingKeys: String, CodingKey {
        case enabled
        case intervalSeconds = "interval_seconds"
        case lastStartedAt = "last_started_at"
        case lastFinishedAt = "last_finished_at"
        case lastStatus = "last_status"
        case lastError = "last_error"
        case lastOutputPath = "last_output_path"
        case lastSizeBytes = "last_size_bytes"
        case nextRunAt = "next_run_at"
    }
}

public struct PeriodicBackupFile: Codable, Equatable {
    public var servers: [String: PeriodicBackupServerState]

    public init(servers: [String: PeriodicBackupServerState] = [:]) {
        self.servers = servers
    }
}

public enum PeriodicBackupStore {
    public static func defaultStatePath() -> String {
        (NSHomeDirectory() as NSString).appendingPathComponent(".ai-critic/periodic-backup.json")
    }

    public static func load(path: String = defaultStatePath()) -> PeriodicBackupFile {
        guard let data = try? Data(contentsOf: URL(fileURLWithPath: path)),
              let decoded = try? JSONDecoder().decode(PeriodicBackupFile.self, from: data) else {
            return PeriodicBackupFile()
        }
        return decoded
    }

    public static func save(_ file: PeriodicBackupFile, path: String = defaultStatePath()) throws {
        let url = URL(fileURLWithPath: path)
        try FileManager.default.createDirectory(
            at: url.deletingLastPathComponent(),
            withIntermediateDirectories: true
        )
        let enc = JSONEncoder()
        enc.outputFormatting = [.prettyPrinted, .sortedKeys]
        let data = try enc.encode(file)
        try data.write(to: url, options: .atomic)
    }

    public static func state(for serverName: String, path: String = defaultStatePath()) -> PeriodicBackupServerState {
        let file = load(path: path)
        // Explicit default: enabled = false (no auto-enable).
        return file.servers[serverName] ?? PeriodicBackupServerState(enabled: false)
    }

    public static func update(
        serverName: String,
        path: String = defaultStatePath(),
        mutate: (inout PeriodicBackupServerState) -> Void
    ) throws {
        var file = load(path: path)
        var st = file.servers[serverName] ?? PeriodicBackupServerState(enabled: false)
        mutate(&st)
        file.servers[serverName] = st
        try save(file, path: path)
    }

    public static func parseRFC3339(_ s: String) -> Date? {
        let trimmed = s.trimmingCharacters(in: .whitespacesAndNewlines)
        guard !trimmed.isEmpty else { return nil }
        let f = ISO8601DateFormatter()
        f.formatOptions = [.withInternetDateTime, .withFractionalSeconds]
        if let d = f.date(from: trimmed) { return d }
        f.formatOptions = [.withInternetDateTime]
        return f.date(from: trimmed)
    }

    public static func formatRFC3339(_ d: Date) -> String {
        let f = ISO8601DateFormatter()
        f.formatOptions = [.withInternetDateTime]
        f.timeZone = TimeZone(secondsFromGMT: 0)
        return f.string(from: d)
    }

    /// Map stored state → display TaskStatus for the Status menu title.
    public static func taskStatus(from st: PeriodicBackupServerState, running: Bool) -> BackupMenuFormatter.TaskStatus {
        if !st.enabled {
            return BackupMenuFormatter.TaskStatus(enabled: false, phase: .off)
        }
        if running {
            return BackupMenuFormatter.TaskStatus(
                enabled: true,
                phase: .running,
                lastFinishedAt: parseRFC3339(st.lastFinishedAt),
                nextRunAt: parseRFC3339(st.nextRunAt),
                lastError: st.lastError,
                lastSizeBytes: st.lastSizeBytes
            )
        }
        if st.lastStatus == "error" || !st.lastError.isEmpty && st.lastStatus != "idle" {
            // Prefer error phase when last run failed.
            if st.lastStatus == "error" {
                return BackupMenuFormatter.TaskStatus(
                    enabled: true,
                    phase: .error,
                    lastFinishedAt: parseRFC3339(st.lastFinishedAt),
                    nextRunAt: parseRFC3339(st.nextRunAt),
                    lastError: st.lastError,
                    lastSizeBytes: st.lastSizeBytes
                )
            }
        }
        return BackupMenuFormatter.TaskStatus(
            enabled: true,
            phase: .idle,
            lastFinishedAt: parseRFC3339(st.lastFinishedAt),
            nextRunAt: parseRFC3339(st.nextRunAt),
            lastError: st.lastError,
            lastSizeBytes: st.lastSizeBytes
        )
    }

    /// List local `.tar.xz` archives under backup dir as FileEntry.
    public static func listBackupEntries(dir: String) -> [BackupMenuFormatter.FileEntry] {
        let fm = FileManager.default
        guard let names = try? fm.contentsOfDirectory(atPath: dir) else { return [] }
        var out: [BackupMenuFormatter.FileEntry] = []
        for name in names where name.hasSuffix(".tar.xz") {
            let path = (dir as NSString).appendingPathComponent(name)
            guard let attrs = try? fm.attributesOfItem(atPath: path),
                  let mod = attrs[.modificationDate] as? Date else { continue }
            let size = (attrs[.size] as? NSNumber)?.int64Value ?? 0
            out.append(BackupMenuFormatter.FileEntry(path: path, modTime: mod, sizeBytes: size))
        }
        return BackupMenuFormatter.sortBackupEntriesNewestFirst(out)
    }

    /// Apply retention prune: delete files outside keep set.
    public static func pruneBackupDir(dir: String, now: Date = Date()) {
        let entries = listBackupEntries(dir: dir)
        let (_, del) = BackupMenuFormatter.pruneBackupFiles(entries, now: now)
        for e in del {
            try? FileManager.default.removeItem(atPath: e.path)
        }
    }
}
