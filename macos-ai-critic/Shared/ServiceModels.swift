import Foundation

public struct ServiceStatus: Decodable, Identifiable, Equatable {
    public let id: String
    public let name: String
    public let status: String
    public let pid: Int
    public let logPath: String
    public let desiredRunning: Bool
    public let enabled: Bool

    public init(
        id: String,
        name: String,
        status: String,
        pid: Int,
        logPath: String,
        desiredRunning: Bool,
        enabled: Bool
    ) {
        self.id = id
        self.name = name
        self.status = status
        self.pid = pid
        self.logPath = logPath
        self.desiredRunning = desiredRunning
        self.enabled = enabled
    }
}

/// Response body for service control APIs.
/// Shapes differ by action (must all decode without error on HTTP 200):
/// - start: full ServiceStatus object (no `message`)
/// - stop/restart: `{"status":"ok"}` (no `message`)
/// - enable/disable: `{status, message, service?}`
public struct ServiceActionResponse: Decodable {
    public let status: String?
    public let message: String?
    public let service: ServiceStatus?

    public init(status: String? = nil, message: String? = nil, service: ServiceStatus? = nil) {
        self.status = status
        self.message = message
        self.service = service
    }

    /// User-facing alert text for enable/disable; empty when server omitted message.
    public var displayMessage: String {
        if let message, !message.isEmpty {
            return message
        }
        if let status, !status.isEmpty {
            return status
        }
        return "OK"
    }
}

public enum ServiceClientError: LocalizedError {
    case unreachable(String)
    case notConfigured

    public var errorDescription: String? {
        switch self {
        case .unreachable(let detail):
            return detail
        case .notConfigured:
            return "Remote server not configured"
        }
    }
}
