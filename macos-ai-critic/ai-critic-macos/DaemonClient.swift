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

    func keepAliveStatus() async throws -> KeepAliveStatus {
        let (data, response) = try await get(path: "/api/keep-alive/status")
        guard let http = response as? HTTPURLResponse, http.statusCode == 200 else {
            throw DaemonClientError.unreachable("keep-alive status request failed")
        }
        return try JSONDecoder().decode(KeepAliveStatus.self, from: data)
    }

    func restartServer() async throws {
        var request = URLRequest(url: URL(string: baseURL + "/api/keep-alive/restart")!)
        request.httpMethod = "POST"
        let (_, response) = try await session.data(for: request)
        guard let http = response as? HTTPURLResponse, http.statusCode == 200 else {
            throw DaemonClientError.unreachable("restart request failed")
        }
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