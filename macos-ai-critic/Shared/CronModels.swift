import Foundation

/// Cron task status from GET /api/cron-tasks and action endpoints.
/// Fields needed for menu titles, actions, and View Logs.
public struct CronTaskStatus: Decodable, Identifiable, Equatable {
    public let id: String
    public let name: String
    public let status: String
    public let enabled: Bool
    public let scheduleMode: String
    public let interval: String
    public let cronExpr: String
    public let logPath: String
    public let pid: Int

    public init(
        id: String,
        name: String,
        status: String,
        enabled: Bool,
        scheduleMode: String,
        interval: String = "",
        cronExpr: String = "",
        logPath: String = "",
        pid: Int = 0
    ) {
        self.id = id
        self.name = name
        self.status = status
        self.enabled = enabled
        self.scheduleMode = scheduleMode
        self.interval = interval
        self.cronExpr = cronExpr
        self.logPath = logPath
        self.pid = pid
    }

    public init(from decoder: Decoder) throws {
        let c = try decoder.container(keyedBy: CodingKeys.self)
        id = try c.decode(String.self, forKey: .id)
        name = try c.decodeIfPresent(String.self, forKey: .name) ?? ""
        status = try c.decodeIfPresent(String.self, forKey: .status) ?? "idle"
        enabled = try c.decodeIfPresent(Bool.self, forKey: .enabled) ?? true
        scheduleMode = try c.decodeIfPresent(String.self, forKey: .scheduleMode) ?? ""
        interval = try c.decodeIfPresent(String.self, forKey: .interval) ?? ""
        cronExpr = try c.decodeIfPresent(String.self, forKey: .cronExpr) ?? ""
        logPath = try c.decodeIfPresent(String.self, forKey: .logPath) ?? ""
        pid = try c.decodeIfPresent(Int.self, forKey: .pid) ?? 0
    }

    private enum CodingKeys: String, CodingKey {
        case id, name, status, enabled, scheduleMode, interval, cronExpr, logPath, pid
    }
}

/// Optional message wrapper for enable/disable alert (server may return bare CronTaskStatus).
public struct CronTaskActionResponse: Decodable {
    public let message: String?
    public let status: String?

    public init(message: String? = nil, status: String? = nil) {
        self.message = message
        self.status = status
    }

    /// User-facing alert text; prefers server message, else Task updated.
    public var displayMessage: String {
        CronMenuFormatter.cronToggleAlertMessage(serverMessage: message ?? "")
    }
}

/// SSE event from GET /api/logs/stream.
public struct LogStreamEvent: Decodable {
    public let type: String
    public let message: String?
    public let status: String?

    public init(type: String, message: String? = nil, status: String? = nil) {
        self.type = type
        self.message = message
        self.status = status
    }
}
