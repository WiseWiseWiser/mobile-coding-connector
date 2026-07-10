import Foundation

/// Create/update body for POST/PUT /api/cron-tasks (menu-bar UI; no extraEnv).
/// `cronExpr` as stored/sent is always UTC.
public struct CronTaskDefinition: Codable, Equatable {
    public var id: String?
    public var name: String
    public var command: String
    public var workingDir: String?
    public var scheduleMode: String
    public var interval: String?
    public var cronExpr: String?
    public var timeout: String?
    public var enabled: Bool?

    public init(
        id: String? = nil,
        name: String,
        command: String,
        workingDir: String? = nil,
        scheduleMode: String,
        interval: String? = nil,
        cronExpr: String? = nil,
        timeout: String? = nil,
        enabled: Bool? = true
    ) {
        self.id = id
        self.name = name
        self.command = command
        self.workingDir = workingDir
        self.scheduleMode = scheduleMode
        self.interval = interval
        self.cronExpr = cronExpr
        self.timeout = timeout
        self.enabled = enabled
    }

    private enum CodingKeys: String, CodingKey {
        case id, name, command, workingDir, scheduleMode, interval, cronExpr, timeout, enabled
    }

    public func encode(to encoder: Encoder) throws {
        var c = encoder.container(keyedBy: CodingKeys.self)
        if let id, !id.isEmpty {
            try c.encode(id, forKey: .id)
        }
        try c.encode(name, forKey: .name)
        try c.encode(command, forKey: .command)
        if let workingDir, !workingDir.isEmpty {
            try c.encode(workingDir, forKey: .workingDir)
        }
        try c.encode(scheduleMode, forKey: .scheduleMode)
        if scheduleMode == "interval", let interval, !interval.isEmpty {
            try c.encode(interval, forKey: .interval)
        }
        if scheduleMode == "cron", let cronExpr, !cronExpr.isEmpty {
            try c.encode(cronExpr, forKey: .cronExpr)
        }
        if let timeout, !timeout.isEmpty {
            try c.encode(timeout, forKey: .timeout)
        }
        if let enabled {
            try c.encode(enabled, forKey: .enabled)
        }
    }
}

/// Cron task status from GET /api/cron-tasks and action endpoints.
/// Includes definition fields needed for Cron Editor prefill.
public struct CronTaskStatus: Decodable, Identifiable, Equatable {
    public let id: String
    public let name: String
    public let status: String
    public let enabled: Bool
    public let scheduleMode: String
    public let interval: String
    public let cronExpr: String
    public let command: String
    public let workingDir: String
    public let timeout: String
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
        command: String = "",
        workingDir: String = "",
        timeout: String = "",
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
        self.command = command
        self.workingDir = workingDir
        self.timeout = timeout
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
        command = try c.decodeIfPresent(String.self, forKey: .command) ?? ""
        workingDir = try c.decodeIfPresent(String.self, forKey: .workingDir) ?? ""
        timeout = try c.decodeIfPresent(String.self, forKey: .timeout) ?? ""
        logPath = try c.decodeIfPresent(String.self, forKey: .logPath) ?? ""
        pid = try c.decodeIfPresent(Int.self, forKey: .pid) ?? 0
    }

    private enum CodingKeys: String, CodingKey {
        case id, name, status, enabled, scheduleMode, interval, cronExpr
        case command, workingDir, timeout, logPath, pid
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
