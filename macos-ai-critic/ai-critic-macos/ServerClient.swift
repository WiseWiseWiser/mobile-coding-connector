import Foundation
import AICriticMacShared

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

// ServiceStatus / ServiceActionResponse live in AICriticMacShared.

struct LogStreamEvent: Decodable {
    let type: String
    let message: String?
    let status: String?
}

enum ServerClientError: LocalizedError {
    case unreachable(String)

    var errorDescription: String? {
        switch self {
        case .unreachable(let detail):
            return detail
        }
    }
}

final class ServerClient {
    static let shared = ServerClient()

    // Main server business plane listens on 23712 by default.
    private let serverPort = 23712
    private let session: URLSession

    init(session: URLSession = .shared) {
        self.session = session
    }

    private var baseURL: String { "http://127.0.0.1:\(serverPort)" }

    func grokUsage() async throws -> GrokUsageResponse {
        let (data, response) = try await get(path: "/api/grok/usage")
        guard let http = response as? HTTPURLResponse, http.statusCode == 200 else {
            throw ServerClientError.unreachable("grok usage request failed")
        }
        return try JSONDecoder().decode(GrokUsageResponse.self, from: data)
    }

    func codexUsage() async throws -> CodexUsageResponse {
        let (data, response) = try await get(path: "/api/codex/usage")
        guard let http = response as? HTTPURLResponse, http.statusCode == 200 else {
            throw ServerClientError.unreachable("codex usage request failed")
        }
        return try JSONDecoder().decode(CodexUsageResponse.self, from: data)
    }

    func listServices() async throws -> [ServiceStatus] {
        let (data, response) = try await get(path: "/api/services?all=1")
        guard let http = response as? HTTPURLResponse, http.statusCode == 200 else {
            throw ServerClientError.unreachable("services list request failed")
        }
        return try JSONDecoder().decode([ServiceStatus].self, from: data)
    }

    /// List terminal sessions via GET /api/terminal/sessions (paginated; all pages).
    func listTerminalSessions() async throws -> [TerminalSession] {
        var page = 1
        var sessions: [TerminalSession] = []
        while true {
            let path = "/api/terminal/sessions?page=\(page)&page_size=100"
            let (data, response) = try await get(path: path)
            guard let http = response as? HTTPURLResponse, http.statusCode == 200 else {
                throw ServerClientError.unreachable("terminal sessions list request failed")
            }
            let decoded = try JSONDecoder().decode(TerminalSessionsPage.self, from: data)
            sessions.append(contentsOf: decoded.sessions)
            if decoded.totalPages <= page || decoded.sessions.isEmpty {
                break
            }
            page += 1
        }
        return sessions
    }

    func startService(id: String) async throws {
        try await postServiceAction(path: "/api/services/start", id: id)
    }

    func stopService(id: String) async throws {
        try await postServiceAction(path: "/api/services/stop", id: id)
    }

    func restartService(id: String) async throws {
        try await postServiceAction(path: "/api/services/restart", id: id)
    }

    func disableService(id: String) async throws -> ServiceActionResponse {
        try await postServiceActionWithResponse(path: "/api/services/disable", id: id)
    }

    func enableService(id: String) async throws -> ServiceActionResponse {
        try await postServiceActionWithResponse(path: "/api/services/enable", id: id)
    }

    func debugLogSettings() async throws -> DebugLogSettings {
        let (data, response) = try await get(path: "/api/debug/log")
        guard let http = response as? HTTPURLResponse, http.statusCode == 200 else {
            throw ServerClientError.unreachable("debug settings request failed")
        }
        return try JSONDecoder().decode(DebugLogSettings.self, from: data)
    }

    func setDebugLog(enabled: Bool) async throws -> DebugLogSettings {
        var request = URLRequest(url: URL(string: baseURL + "/api/debug/log")!)
        request.httpMethod = "PUT"
        request.setValue("application/json", forHTTPHeaderField: "Content-Type")
        request.httpBody = try JSONEncoder().encode(["enabled": enabled])
        let (data, response) = try await session.data(for: request)
        guard let http = response as? HTTPURLResponse, http.statusCode == 200 else {
            throw ServerClientError.unreachable("debug settings update failed")
        }
        return try JSONDecoder().decode(DebugLogSettings.self, from: data)
    }

    func isHealthy() async -> Bool {
        do {
            let (_, response) = try await get(path: "/ping")
            guard let http = response as? HTTPURLResponse else { return false }
            return http.statusCode == 200
        } catch {
            return false
        }
    }

    func streamLog(path: String, lines: Int = 1000) -> AsyncThrowingStream<LogStreamEvent, Error> {
        AsyncThrowingStream { continuation in
            let task = Task {
                do {
                    let trimmed = path.trimmingCharacters(in: .whitespacesAndNewlines)
                    guard !trimmed.isEmpty else {
                        throw ServerClientError.unreachable("log path is required")
                    }

                    var components = URLComponents(string: baseURL + "/api/logs/stream")!
                    components.queryItems = [
                        URLQueryItem(name: "path", value: trimmed),
                        URLQueryItem(name: "lines", value: String(lines > 0 ? lines : 1000)),
                    ]
                    guard let url = components.url else {
                        throw ServerClientError.unreachable("invalid url")
                    }

                    var request = URLRequest(url: url)
                    request.setValue("text/event-stream", forHTTPHeaderField: "Accept")

                    let (bytes, response) = try await session.bytes(for: request)
                    guard let http = response as? HTTPURLResponse, (200..<300).contains(http.statusCode) else {
                        throw ServerClientError.unreachable("log stream request failed")
                    }

                    for try await line in bytes.lines {
                        try Task.checkCancellation()
                        let trimmedLine = line.trimmingCharacters(in: .whitespacesAndNewlines)
                        guard !trimmedLine.isEmpty, trimmedLine.hasPrefix("data: ") else { continue }

                        let payload = String(trimmedLine.dropFirst("data: ".count))
                        let data = Data(payload.utf8)
                        let event = try JSONDecoder().decode(LogStreamEvent.self, from: data)
                        continuation.yield(event)
                        if event.type == "error" {
                            throw ServerClientError.unreachable(event.message ?? "log stream failed")
                        }
                    }
                    continuation.finish()
                } catch {
                    continuation.finish(throwing: error)
                }
            }
            continuation.onTermination = { _ in
                task.cancel()
            }
        }
    }

    private func get(path: String) async throws -> (Data, URLResponse) {
        guard let url = URL(string: baseURL + path) else {
            throw ServerClientError.unreachable("invalid url")
        }
        return try await session.data(from: url)
    }

    private func postServiceAction(path: String, id: String) async throws {
        _ = try await postServiceActionWithResponse(path: path, id: id)
    }

    private func postServiceActionWithResponse(path: String, id: String) async throws -> ServiceActionResponse {
        guard let url = URL(string: baseURL + path + "?id=" + id.addingPercentEncoding(withAllowedCharacters: .urlQueryAllowed)!) else {
            throw ServerClientError.unreachable("invalid url")
        }
        var request = URLRequest(url: url)
        request.httpMethod = "POST"
        let (data, response) = try await session.data(for: request)
        guard let http = response as? HTTPURLResponse, http.statusCode == 200 else {
            throw ServerClientError.unreachable("service action failed")
        }
        return try JSONDecoder().decode(ServiceActionResponse.self, from: data)
    }
}