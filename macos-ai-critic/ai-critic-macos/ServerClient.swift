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

// ServiceStatus / ServiceActionResponse / CronTaskStatus / LogStreamEvent live in AICriticMacShared.

// MARK: - wrk projects / worktrees (GET/POST /api/wrk/*)

struct WrkWorktreeStatus: Decodable, Identifiable {
    var id: String { path }
    let path: String
    let name: String
    let branch: String?
    let clean: Bool
    let isMain: Bool
    let error: String?

    enum CodingKeys: String, CodingKey {
        case path, name, branch, clean, error
        case isMain = "is_main"
    }
}

struct WrkProjectStatus: Decodable, Identifiable {
    var id: String { path }
    let path: String
    let name: String
    let branch: String?
    let commit: String?
    let subject: String?
    let clean: Bool
    let error: String?
    let worktrees: [WrkWorktreeStatus]?
}

struct WrkListProjectsResponse: Decodable {
    let projects: [WrkProjectStatus]
}

struct WrkCreateWorktreeRequest: Encodable {
    let projectPath: String
    let task: String?

    enum CodingKeys: String, CodingKey {
        case projectPath = "project_path"
        case task
    }
}

struct WrkCreateWorktreeResponse: Decodable {
    let path: String
    let branch: String
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
    /// Resolved Bearer token (config → credentials → empty). Applied on all API requests.
    private var authToken: String

    init(session: URLSession = .shared) {
        self.session = session
        self.authToken = LocalAuth.resolveLocalServerToken().token
    }

    /// Re-read token from disk (e.g. after credentials change).
    func refreshAuthToken() {
        authToken = LocalAuth.resolveLocalServerToken().token
    }

    private var baseURL: String { "http://127.0.0.1:\(serverPort)" }

    /// Apply Authorization: Bearer <token> when token is non-empty; omit when empty.
    private func applyAuth(_ request: inout URLRequest) {
        let header = LocalAuth.authorizationHeader(token: authToken)
        if !header.isEmpty {
            request.setValue(header, forHTTPHeaderField: "Authorization")
        }
    }

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

    /// List cron tasks via GET /api/cron-tasks (no all=1).
    func listCronTasks() async throws -> [CronTaskStatus] {
        let (data, response) = try await get(path: "/api/cron-tasks")
        guard let http = response as? HTTPURLResponse, http.statusCode == 200 else {
            throw ServerClientError.unreachable("cron tasks list request failed")
        }
        return try JSONDecoder().decode([CronTaskStatus].self, from: data)
    }

    func runCronTask(id: String) async throws {
        _ = try await postCronActionWithResponse(path: "/api/cron-tasks/run", id: id)
    }

    func enableCronTask(id: String) async throws -> CronTaskActionResponse {
        try await postCronActionWithResponse(path: "/api/cron-tasks/enable", id: id)
    }

    func disableCronTask(id: String) async throws -> CronTaskActionResponse {
        try await postCronActionWithResponse(path: "/api/cron-tasks/disable", id: id)
    }

    /// POST /api/cron-tasks — createCronTask.
    func createCronTask(_ def: CronTaskDefinition) async throws -> CronTaskStatus {
        try await saveCronTask(method: "POST", def: def)
    }

    /// PUT /api/cron-tasks — updateCronTask (id required in body).
    func updateCronTask(_ def: CronTaskDefinition) async throws -> CronTaskStatus {
        guard let id = def.id?.trimmingCharacters(in: .whitespacesAndNewlines), !id.isEmpty else {
            throw ServerClientError.unreachable("task id is required")
        }
        return try await saveCronTask(method: "PUT", def: def)
    }

    /// DELETE /api/cron-tasks?id=
    func deleteCronTask(id: String) async throws {
        let trimmed = id.trimmingCharacters(in: .whitespacesAndNewlines)
        guard !trimmed.isEmpty else {
            throw ServerClientError.unreachable("task id is required")
        }
        guard let encoded = trimmed.addingPercentEncoding(withAllowedCharacters: .urlQueryAllowed),
              let url = URL(string: baseURL + "/api/cron-tasks?id=" + encoded) else {
            throw ServerClientError.unreachable("invalid url")
        }
        var request = URLRequest(url: url)
        request.httpMethod = "DELETE"
        applyAuth(&request)
        let (_, response) = try await session.data(for: request)
        guard let http = response as? HTTPURLResponse, (200..<300).contains(http.statusCode) else {
            throw ServerClientError.unreachable("delete cron task failed")
        }
    }

    private func saveCronTask(method: String, def: CronTaskDefinition) async throws -> CronTaskStatus {
        guard let url = URL(string: baseURL + "/api/cron-tasks") else {
            throw ServerClientError.unreachable("invalid url")
        }
        var request = URLRequest(url: url)
        request.httpMethod = method
        request.setValue("application/json", forHTTPHeaderField: "Content-Type")
        applyAuth(&request)
        request.httpBody = try JSONEncoder().encode(def)
        let (data, response) = try await session.data(for: request)
        guard let http = response as? HTTPURLResponse, (200..<300).contains(http.statusCode) else {
            throw ServerClientError.unreachable("save cron task failed")
        }
        return try JSONDecoder().decode(CronTaskStatus.self, from: data)
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

    /// List wrk projects via GET /api/wrk/projects.
    func listWrkProjects() async throws -> [WrkProjectStatus] {
        let (data, response) = try await get(path: "/api/wrk/projects")
        guard let http = response as? HTTPURLResponse, http.statusCode == 200 else {
            throw ServerClientError.unreachable("wrk projects list request failed")
        }
        let decoded = try JSONDecoder().decode(WrkListProjectsResponse.self, from: data)
        return decoded.projects
    }

    /// Create a worktree via POST /api/wrk/worktrees.
    /// Empty/whitespace task is sent as omitted (no slug).
    func createWrkWorktree(projectPath: String, task: String? = nil) async throws -> WrkCreateWorktreeResponse {
        guard let url = URL(string: baseURL + "/api/wrk/worktrees") else {
            throw ServerClientError.unreachable("invalid url")
        }
        var request = URLRequest(url: url)
        request.httpMethod = "POST"
        request.setValue("application/json", forHTTPHeaderField: "Content-Type")
        applyAuth(&request)
        let trimmedTask = task?.trimmingCharacters(in: .whitespacesAndNewlines)
        let body = WrkCreateWorktreeRequest(
            projectPath: projectPath,
            task: (trimmedTask?.isEmpty == false) ? trimmedTask : nil
        )
        request.httpBody = try JSONEncoder().encode(body)
        let (data, response) = try await session.data(for: request)
        guard let http = response as? HTTPURLResponse, http.statusCode == 200 else {
            throw ServerClientError.unreachable("wrk create worktree request failed")
        }
        return try JSONDecoder().decode(WrkCreateWorktreeResponse.self, from: data)
    }

    /// Open a directory in iTerm2 via POST /api/local/iterm2/open.
    /// - Parameters:
    ///   - dir: Absolute path to open (required on server).
    ///   - mode: Optional `"reuse"` | `"new"` | `"smart"`; omit for server default reuse.
    ///   - send: Optional follow-up shell commands after `cd`.
    func openITerm2(dir: String, mode: String? = nil, send: [String]? = nil) async throws {
        guard let url = URL(string: baseURL + "/api/local/iterm2/open") else {
            throw ServerClientError.unreachable("invalid url")
        }
        var request = URLRequest(url: url)
        request.httpMethod = "POST"
        request.setValue("application/json", forHTTPHeaderField: "Content-Type")
        applyAuth(&request)
        var body: [String: Any] = ["dir": dir]
        if let mode, !mode.isEmpty {
            body["mode"] = mode
        }
        if let send {
            body["send"] = send
        }
        request.httpBody = try JSONSerialization.data(withJSONObject: body)
        let (_, response) = try await session.data(for: request)
        guard let http = response as? HTTPURLResponse, http.statusCode == 200 else {
            throw ServerClientError.unreachable("iterm2 open request failed")
        }
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
        applyAuth(&request)
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
                    self.applyAuth(&request)

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
        var request = URLRequest(url: url)
        request.httpMethod = "GET"
        applyAuth(&request)
        return try await session.data(for: request)
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
        applyAuth(&request)
        let (data, response) = try await session.data(for: request)
        guard let http = response as? HTTPURLResponse, http.statusCode == 200 else {
            throw ServerClientError.unreachable("service action failed")
        }
        return try JSONDecoder().decode(ServiceActionResponse.self, from: data)
    }

    private func postCronActionWithResponse(path: String, id: String) async throws -> CronTaskActionResponse {
        guard let url = URL(string: baseURL + path + "?id=" + id.addingPercentEncoding(withAllowedCharacters: .urlQueryAllowed)!) else {
            throw ServerClientError.unreachable("invalid url")
        }
        var request = URLRequest(url: url)
        request.httpMethod = "POST"
        let (data, response) = try await session.data(for: request)
        guard let http = response as? HTTPURLResponse, http.statusCode == 200 else {
            throw ServerClientError.unreachable("cron action failed")
        }
        return try ServiceClient.decodeCronActionBody(data)
    }
}