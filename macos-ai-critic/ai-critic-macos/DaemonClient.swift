import Foundation

struct GrokUsageResponse: Decodable {
    let status: String
    let weeklyLimit: String?
    let nextReset: String?
    let error: String?
    let updatedAt: String?

    enum CodingKeys: String, CodingKey {
        case status
        case weeklyLimit = "weekly_limit"
        case nextReset = "next_reset"
        case error
        case updatedAt = "updated_at"
    }
}

struct CodexUsageResponse: Decodable {
    let status: String
    let monthlyUsage: String?
    let creditsUsed: String?
    let creditsTotal: String?
    let nextReset: String?
    let error: String?
    let updatedAt: String?

    enum CodingKeys: String, CodingKey {
        case status
        case monthlyUsage = "monthly_usage"
        case creditsUsed = "credits_used"
        case creditsTotal = "credits_total"
        case nextReset = "next_reset"
        case error
        case updatedAt = "updated_at"
    }
}

struct DebugLogSettings: Codable {
    let enabled: Bool
    let path: String
}

private struct DebugLogSettingsRequest: Encodable {
    let enabled: Bool
}

struct KeepAliveStatus: Decodable {
    let running: Bool
    let serverPort: Int
    let serverPID: Int
    let keepAlivePort: Int
    let keepAlivePID: Int

    enum CodingKeys: String, CodingKey {
        case running
        case serverPort = "server_port"
        case serverPID = "server_pid"
        case keepAlivePort = "keep_alive_port"
        case keepAlivePID = "keep_alive_pid"
    }
}

enum DaemonClientError: LocalizedError {
    case unreachable(String)

    var errorDescription: String? {
        switch self {
        case .unreachable(let detail):
            return detail
        }
    }
}

final class DaemonClient {
    static let shared = DaemonClient()

    private let keepAlivePort = 23312
    private let session: URLSession

    init(session: URLSession = .shared) {
        self.session = session
    }

    private var baseURL: String { "http://127.0.0.1:\(keepAlivePort)" }

    func grokUsage() async throws -> GrokUsageResponse {
        let (data, response) = try await get(path: "/api/grok/usage")
        guard let http = response as? HTTPURLResponse, http.statusCode == 200 else {
            throw DaemonClientError.unreachable("grok usage request failed")
        }
        return try JSONDecoder().decode(GrokUsageResponse.self, from: data)
    }

    func codexUsage() async throws -> CodexUsageResponse {
        let (data, response) = try await get(path: "/api/codex/usage")
        guard let http = response as? HTTPURLResponse, http.statusCode == 200 else {
            throw DaemonClientError.unreachable("codex usage request failed")
        }
        return try JSONDecoder().decode(CodexUsageResponse.self, from: data)
    }

    func keepAliveStatus() async throws -> KeepAliveStatus {
        let (data, response) = try await get(path: "/api/keep-alive/status")
        guard let http = response as? HTTPURLResponse, http.statusCode == 200 else {
            throw DaemonClientError.unreachable("keep-alive status request failed")
        }
        return try JSONDecoder().decode(KeepAliveStatus.self, from: data)
    }

    func restartDaemon() async throws {
        var request = URLRequest(url: URL(string: baseURL + "/api/keep-alive/restart-daemon")!)
        request.httpMethod = "POST"
        request.setValue("text/event-stream", forHTTPHeaderField: "Accept")
        let (_, response) = try await session.data(for: request)
        guard let http = response as? HTTPURLResponse, http.statusCode == 200 else {
            throw DaemonClientError.unreachable("restart daemon request failed")
        }
    }

    func debugLogSettings() async throws -> DebugLogSettings {
        let (data, response) = try await get(path: "/api/keep-alive/debug")
        guard let http = response as? HTTPURLResponse, http.statusCode == 200 else {
            throw DaemonClientError.unreachable("debug settings request failed")
        }
        return try JSONDecoder().decode(DebugLogSettings.self, from: data)
    }

    func setDebugLog(enabled: Bool) async throws -> DebugLogSettings {
        var request = URLRequest(url: URL(string: baseURL + "/api/keep-alive/debug")!)
        request.httpMethod = "PUT"
        request.setValue("application/json", forHTTPHeaderField: "Content-Type")
        request.httpBody = try JSONEncoder().encode(DebugLogSettingsRequest(enabled: enabled))
        let (data, response) = try await session.data(for: request)
        guard let http = response as? HTTPURLResponse, http.statusCode == 200 else {
            throw DaemonClientError.unreachable("debug settings update failed")
        }
        return try JSONDecoder().decode(DebugLogSettings.self, from: data)
    }

    func isHealthy() async -> Bool {
        do {
            let status = try await keepAliveStatus()
            return status.keepAlivePID > 0
        } catch {
            return false
        }
    }

    private func get(path: String) async throws -> (Data, URLResponse) {
        guard let url = URL(string: baseURL + path) else {
            throw DaemonClientError.unreachable("invalid url")
        }
        return try await session.data(from: url)
    }
}