import Foundation

/// Downloads remote machine backups via stream + `archive_token` (same path as CLI).
///
/// Flow:
/// 1. POST `/api/remote-agent/machine/backup/stream` with `{"archive":true,...}`
/// 2. Read SSE frames until `done` with `archive_token`
/// 3. GET `/api/remote-agent/machine/backup/archive?token=…` and write local `.tar.xz`
public final class MachineBackupClient: @unchecked Sendable {
    public var baseURL: String
    public var token: String
    private let session: URLSession

    public init(baseURL: String = "", token: String = "", session: URLSession = .shared) {
        self.baseURL = ServiceClient.normalizeBaseURL(baseURL)
        self.token = token
        self.session = session
    }

    public func configure(baseURL: String, token: String) {
        self.baseURL = ServiceClient.normalizeBaseURL(baseURL)
        self.token = token
    }

    public var isConfigured: Bool { !baseURL.isEmpty }

    public static let backupStreamPath = "/api/remote-agent/machine/backup/stream"
    public static let backupArchivePath = "/api/remote-agent/machine/backup/archive"

    /// Stream backup with archive=true, extract archiveToken from done frame, download archive to destPath.
    @discardableResult
    public func downloadBackupArchive(to destPath: String) async throws -> Int64 {
        guard isConfigured else { throw ServiceClientError.notConfigured }
        let archiveToken = try await streamBackupForArchiveToken()
        return try await downloadArchiveByToken(archiveToken, to: destPath)
    }

    /// POST machine/backup/stream and return archive_token from the done SSE frame.
    public func streamBackupForArchiveToken() async throws -> String {
        guard isConfigured else { throw ServiceClientError.notConfigured }
        guard let url = URL(string: baseURL + Self.backupStreamPath) else {
            throw ServiceClientError.unreachable("invalid backup stream url")
        }
        var request = URLRequest(url: url)
        request.httpMethod = "POST"
        request.setValue("application/json", forHTTPHeaderField: "Content-Type")
        request.setValue("text/event-stream", forHTTPHeaderField: "Accept")
        applyAuth(&request)
        // archive:true packs the archive and returns archive_token in the done frame.
        let body: [String: Any] = [
            "archive": true,
            "exclude": [String](),
            "include": [String](),
        ]
        request.httpBody = try JSONSerialization.data(withJSONObject: body)

        let (bytes, response) = try await session.bytes(for: request)
        guard let http = response as? HTTPURLResponse, (200..<300).contains(http.statusCode) else {
            throw ServiceClientError.unreachable("backup stream request failed")
        }

        var archiveToken: String?
        for try await line in bytes.lines {
            try Task.checkCancellation()
            let trimmed = line.trimmingCharacters(in: .whitespacesAndNewlines)
            guard !trimmed.isEmpty, trimmed.hasPrefix("data: ") else { continue }
            let payload = String(trimmed.dropFirst("data: ".count))
            guard let data = payload.data(using: .utf8),
                  let obj = try? JSONSerialization.jsonObject(with: data) as? [String: Any],
                  let type = obj["type"] as? String else {
                continue
            }
            if type == "error" {
                let msg = (obj["message"] as? String) ?? "backup stream error"
                throw ServiceClientError.unreachable(msg)
            }
            if type == "done" {
                // Accept snake_case archive_token and camelCase archiveToken.
                if let t = obj["archive_token"] as? String, !t.trimmingCharacters(in: .whitespacesAndNewlines).isEmpty {
                    archiveToken = t
                } else if let t = obj["archiveToken"] as? String, !t.trimmingCharacters(in: .whitespacesAndNewlines).isEmpty {
                    archiveToken = t
                }
                break
            }
        }
        guard let token = archiveToken?.trimmingCharacters(in: .whitespacesAndNewlines), !token.isEmpty else {
            throw ServiceClientError.unreachable("backup stream missing archive_token")
        }
        return token
    }

    /// GET archive by token and write to destPath; returns bytes written.
    public func downloadArchiveByToken(_ archiveToken: String, to destPath: String) async throws -> Int64 {
        let trimmed = archiveToken.trimmingCharacters(in: .whitespacesAndNewlines)
        guard !trimmed.isEmpty else {
            throw ServiceClientError.unreachable("archive token is required")
        }
        var components = URLComponents(string: baseURL + Self.backupArchivePath)
        components?.queryItems = [URLQueryItem(name: "token", value: trimmed)]
        guard let url = components?.url else {
            throw ServiceClientError.unreachable("invalid archive download url")
        }
        var request = URLRequest(url: url)
        request.httpMethod = "GET"
        applyAuth(&request)

        let (tempURL, response) = try await session.download(for: request)
        guard let http = response as? HTTPURLResponse, (200..<300).contains(http.statusCode) else {
            throw ServiceClientError.unreachable("archive download failed")
        }

        let dest = URL(fileURLWithPath: destPath)
        let dir = dest.deletingLastPathComponent()
        try FileManager.default.createDirectory(at: dir, withIntermediateDirectories: true)
        // Atomic-ish replace: remove existing, move temp into place.
        if FileManager.default.fileExists(atPath: destPath) {
            try FileManager.default.removeItem(at: dest)
        }
        try FileManager.default.moveItem(at: tempURL, to: dest)
        let attrs = try FileManager.default.attributesOfItem(atPath: destPath)
        let size = (attrs[.size] as? NSNumber)?.int64Value ?? 0
        return size
    }

    private func applyAuth(_ request: inout URLRequest) {
        let header = ServiceClient.authorizationHeader(token: token)
        if !header.isEmpty {
            request.setValue(header, forHTTPHeaderField: "Authorization")
        }
    }
}
