import Foundation

/// HTTP client for service list/control APIs with configurable base URL + Bearer token.
/// Paths mirror `macosapp/serviceapi` (list: GET /api/services?all=1; actions: POST /api/services/{action}?id=).
public final class ServiceClient: @unchecked Sendable {
    public var baseURL: String
    public var token: String
    private let session: URLSession

    public init(baseURL: String = "", token: String = "", session: URLSession = .shared) {
        self.baseURL = Self.normalizeBaseURL(baseURL)
        self.token = token
        self.session = session
    }

    public func configure(baseURL: String, token: String) {
        self.baseURL = Self.normalizeBaseURL(baseURL)
        self.token = token
    }

    public var isConfigured: Bool {
        !baseURL.isEmpty
    }

    public static func normalizeBaseURL(_ base: String) -> String {
        var s = base.trimmingCharacters(in: .whitespacesAndNewlines)
        while s.hasSuffix("/") {
            s.removeLast()
        }
        return s
    }

    /// Pure request plan — same contract as Go `serviceapi.BuildListServicesRequest`.
    public static func buildListServicesRequest(baseURL: String, token: String) throws -> URLRequest {
        let base = normalizeBaseURL(baseURL)
        guard !base.isEmpty else { throw ServiceClientError.notConfigured }
        guard let url = URL(string: base + listServicesPath) else {
            throw ServiceClientError.unreachable("invalid list services url")
        }
        var request = URLRequest(url: url)
        request.httpMethod = "GET"
        applyAuth(&request, token: token)
        return request
    }

    /// Pure request plan — same contract as Go `serviceapi.BuildServiceActionRequest`.
    public static func buildServiceActionRequest(
        baseURL: String,
        token: String,
        action: String,
        id: String
    ) throws -> URLRequest {
        let base = normalizeBaseURL(baseURL)
        guard !base.isEmpty else { throw ServiceClientError.notConfigured }
        let trimmedID = id.trimmingCharacters(in: .whitespacesAndNewlines)
        guard !trimmedID.isEmpty else {
            throw ServiceClientError.unreachable("service id is required")
        }
        let allowed = ["start", "stop", "restart", "enable", "disable"]
        guard allowed.contains(action) else {
            throw ServiceClientError.unreachable("unknown service action")
        }
        var components = URLComponents(string: base + "/api/services/\(action)")
        components?.queryItems = [URLQueryItem(name: "id", value: trimmedID)]
        guard let url = components?.url else {
            throw ServiceClientError.unreachable("invalid service action url")
        }
        var request = URLRequest(url: url)
        request.httpMethod = "POST"
        applyAuth(&request, token: token)
        return request
    }

    public static let listServicesPath = "/api/services?all=1"

    public static func authorizationHeader(token: String) -> String {
        token.isEmpty ? "" : "Bearer \(token)"
    }

    private static func applyAuth(_ request: inout URLRequest, token: String) {
        let header = authorizationHeader(token: token)
        if !header.isEmpty {
            request.setValue(header, forHTTPHeaderField: "Authorization")
        }
    }

    public func listServices() async throws -> [ServiceStatus] {
        let request = try Self.buildListServicesRequest(baseURL: baseURL, token: token)
        let (data, response) = try await session.data(for: request)
        guard let http = response as? HTTPURLResponse, http.statusCode == 200 else {
            throw ServiceClientError.unreachable("services list request failed")
        }
        return try JSONDecoder().decode([ServiceStatus].self, from: data)
    }

    /// List terminal sessions via GET /api/terminal/sessions (paginated; all pages).
    public func listTerminalSessions() async throws -> [TerminalSession] {
        guard isConfigured else { throw ServiceClientError.notConfigured }
        var page = 1
        var sessions: [TerminalSession] = []
        while true {
            var components = URLComponents(string: baseURL + "/api/terminal/sessions")
            components?.queryItems = [
                URLQueryItem(name: "page", value: String(page)),
                URLQueryItem(name: "page_size", value: "100"),
            ]
            guard let url = components?.url else {
                throw ServiceClientError.unreachable("invalid terminal sessions url")
            }
            var request = URLRequest(url: url)
            request.httpMethod = "GET"
            Self.applyAuth(&request, token: token)
            let (data, response) = try await session.data(for: request)
            guard let http = response as? HTTPURLResponse, http.statusCode == 200 else {
                throw ServiceClientError.unreachable("terminal sessions list request failed")
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

    public func startService(id: String) async throws {
        try await postServiceAction(action: "start", id: id)
    }

    public func stopService(id: String) async throws {
        try await postServiceAction(action: "stop", id: id)
    }

    public func restartService(id: String) async throws {
        try await postServiceAction(action: "restart", id: id)
    }

    public func enableService(id: String) async throws -> ServiceActionResponse {
        try await postServiceActionWithResponse(action: "enable", id: id)
    }

    public func disableService(id: String) async throws -> ServiceActionResponse {
        try await postServiceActionWithResponse(action: "disable", id: id)
    }

    // MARK: - Cron tasks (paths mirror macosapp/cronapi)

    public static let listCronTasksPath = "/api/cron-tasks"

    /// Pure request plan — same contract as Go `cronapi.BuildListCronTasksRequest`.
    public static func buildListCronTasksRequest(baseURL: String, token: String) throws -> URLRequest {
        let base = normalizeBaseURL(baseURL)
        guard !base.isEmpty else { throw ServiceClientError.notConfigured }
        guard let url = URL(string: base + listCronTasksPath) else {
            throw ServiceClientError.unreachable("invalid list cron tasks url")
        }
        var request = URLRequest(url: url)
        request.httpMethod = "GET"
        applyAuth(&request, token: token)
        return request
    }

    /// Pure request plan — same contract as Go `cronapi.BuildCronActionRequest`.
    public static func buildCronActionRequest(
        baseURL: String,
        token: String,
        action: String,
        id: String
    ) throws -> URLRequest {
        let base = normalizeBaseURL(baseURL)
        guard !base.isEmpty else { throw ServiceClientError.notConfigured }
        let trimmedID = id.trimmingCharacters(in: .whitespacesAndNewlines)
        guard !trimmedID.isEmpty else {
            throw ServiceClientError.unreachable("task id is required")
        }
        let allowed = ["run", "enable", "disable"]
        guard allowed.contains(action) else {
            throw ServiceClientError.unreachable("unknown cron action")
        }
        var components = URLComponents(string: base + "/api/cron-tasks/\(action)")
        components?.queryItems = [URLQueryItem(name: "id", value: trimmedID)]
        guard let url = components?.url else {
            throw ServiceClientError.unreachable("invalid cron action url")
        }
        var request = URLRequest(url: url)
        request.httpMethod = "POST"
        applyAuth(&request, token: token)
        return request
    }

    public func listCronTasks() async throws -> [CronTaskStatus] {
        let request = try Self.buildListCronTasksRequest(baseURL: baseURL, token: token)
        let (data, response) = try await session.data(for: request)
        guard let http = response as? HTTPURLResponse, http.statusCode == 200 else {
            throw ServiceClientError.unreachable("cron tasks list request failed")
        }
        return try JSONDecoder().decode([CronTaskStatus].self, from: data)
    }

    public func runCronTask(id: String) async throws {
        _ = try await postCronAction(action: "run", id: id)
    }

    public func enableCronTask(id: String) async throws -> CronTaskActionResponse {
        try await postCronAction(action: "enable", id: id)
    }

    public func disableCronTask(id: String) async throws -> CronTaskActionResponse {
        try await postCronAction(action: "disable", id: id)
    }

    /// POST /api/cron-tasks — createCronTask.
    public func createCronTask(_ def: CronTaskDefinition) async throws -> CronTaskStatus {
        try await saveCronTask(method: "POST", def: def)
    }

    /// PUT /api/cron-tasks — updateCronTask (id required in body).
    public func updateCronTask(_ def: CronTaskDefinition) async throws -> CronTaskStatus {
        guard let id = def.id?.trimmingCharacters(in: .whitespacesAndNewlines), !id.isEmpty else {
            throw ServiceClientError.unreachable("task id is required")
        }
        return try await saveCronTask(method: "PUT", def: def)
    }

    /// DELETE /api/cron-tasks?id=
    public func deleteCronTask(id: String) async throws {
        let trimmed = id.trimmingCharacters(in: .whitespacesAndNewlines)
        guard !trimmed.isEmpty else {
            throw ServiceClientError.unreachable("task id is required")
        }
        guard isConfigured else { throw ServiceClientError.notConfigured }
        var components = URLComponents(string: baseURL + Self.listCronTasksPath)
        components?.queryItems = [URLQueryItem(name: "id", value: trimmed)]
        guard let url = components?.url else {
            throw ServiceClientError.unreachable("invalid delete cron task url")
        }
        var request = URLRequest(url: url)
        request.httpMethod = "DELETE"
        Self.applyAuth(&request, token: token)
        let (_, response) = try await session.data(for: request)
        guard let http = response as? HTTPURLResponse, (200..<300).contains(http.statusCode) else {
            throw ServiceClientError.unreachable("delete cron task failed")
        }
    }

    private func saveCronTask(method: String, def: CronTaskDefinition) async throws -> CronTaskStatus {
        guard isConfigured else { throw ServiceClientError.notConfigured }
        guard let url = URL(string: baseURL + Self.listCronTasksPath) else {
            throw ServiceClientError.unreachable("invalid save cron task url")
        }
        var request = URLRequest(url: url)
        request.httpMethod = method
        request.setValue("application/json", forHTTPHeaderField: "Content-Type")
        Self.applyAuth(&request, token: token)
        request.httpBody = try JSONEncoder().encode(def)
        let (data, response) = try await session.data(for: request)
        guard let http = response as? HTTPURLResponse, (200..<300).contains(http.statusCode) else {
            throw ServiceClientError.unreachable("save cron task failed")
        }
        return try JSONDecoder().decode(CronTaskStatus.self, from: data)
    }

    private func postCronAction(action: String, id: String) async throws -> CronTaskActionResponse {
        let request = try Self.buildCronActionRequest(
            baseURL: baseURL,
            token: token,
            action: action,
            id: id
        )
        let (data, response) = try await session.data(for: request)
        guard let http = response as? HTTPURLResponse, http.statusCode == 200 else {
            throw ServiceClientError.unreachable("cron action failed")
        }
        return try Self.decodeCronActionBody(data)
    }

    /// Decode enable/disable/run body; bare CronTaskStatus has no message → client fallback.
    public static func decodeCronActionBody(_ data: Data) throws -> CronTaskActionResponse {
        if data.isEmpty {
            return CronTaskActionResponse(status: "ok")
        }
        if let decoded = try? JSONDecoder().decode(CronTaskActionResponse.self, from: data),
           decoded.message != nil {
            return decoded
        }
        // Bare CronTaskStatus from run/enable/disable — success without message.
        if (try? JSONDecoder().decode(CronTaskStatus.self, from: data)) != nil {
            return CronTaskActionResponse(status: "ok")
        }
        if let decoded = try? JSONDecoder().decode(CronTaskActionResponse.self, from: data) {
            return decoded
        }
        throw ServiceClientError.unreachable("cron action response decode failed")
    }

    /// Stream logs via GET /api/logs/stream?path=&lines= with optional Bearer auth.
    public func streamLog(path: String, lines: Int = 1000) -> AsyncThrowingStream<LogStreamEvent, Error> {
        AsyncThrowingStream { continuation in
            let task = Task {
                do {
                    let trimmed = path.trimmingCharacters(in: .whitespacesAndNewlines)
                    guard !trimmed.isEmpty else {
                        throw ServiceClientError.unreachable("log path is required")
                    }
                    guard isConfigured else {
                        throw ServiceClientError.notConfigured
                    }

                    var components = URLComponents(string: baseURL + "/api/logs/stream")!
                    components.queryItems = [
                        URLQueryItem(name: "path", value: trimmed),
                        URLQueryItem(name: "lines", value: String(lines > 0 ? lines : 1000)),
                    ]
                    guard let url = components.url else {
                        throw ServiceClientError.unreachable("invalid url")
                    }

                    var request = URLRequest(url: url)
                    request.setValue("text/event-stream", forHTTPHeaderField: "Accept")
                    Self.applyAuth(&request, token: token)

                    let (bytes, response) = try await session.bytes(for: request)
                    guard let http = response as? HTTPURLResponse, (200..<300).contains(http.statusCode) else {
                        throw ServiceClientError.unreachable("log stream request failed")
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
                            throw ServiceClientError.unreachable(event.message ?? "log stream failed")
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

    /// start/stop/restart: success is HTTP 200 only (body is ServiceStatus or {"status":"ok"}).
    private func postServiceAction(action: String, id: String) async throws {
        let request = try Self.buildServiceActionRequest(
            baseURL: baseURL,
            token: token,
            action: action,
            id: id
        )
        let (data, response) = try await session.data(for: request)
        guard let http = response as? HTTPURLResponse, http.statusCode == 200 else {
            throw ServiceClientError.unreachable("service action failed")
        }
        // Accept any of the server body shapes; must not throw after 200.
        _ = try Self.decodeServiceActionBody(data)
    }

    private func postServiceActionWithResponse(action: String, id: String) async throws -> ServiceActionResponse {
        let request = try Self.buildServiceActionRequest(
            baseURL: baseURL,
            token: token,
            action: action,
            id: id
        )
        let (data, response) = try await session.data(for: request)
        guard let http = response as? HTTPURLResponse, http.statusCode == 200 else {
            throw ServiceClientError.unreachable("service action failed")
        }
        return try Self.decodeServiceActionBody(data)
    }

    /// Decode enable/disable response or tolerate start/stop/restart bodies.
    public static func decodeServiceActionBody(_ data: Data) throws -> ServiceActionResponse {
        if data.isEmpty {
            return ServiceActionResponse(status: "ok")
        }
        if let decoded = try? JSONDecoder().decode(ServiceActionResponse.self, from: data) {
            return decoded
        }
        // start returns a bare ServiceStatus — treat as success without message.
        if (try? JSONDecoder().decode(ServiceStatus.self, from: data)) != nil {
            return ServiceActionResponse(status: "ok")
        }
        throw ServiceClientError.unreachable("service action response decode failed")
    }
}
