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
