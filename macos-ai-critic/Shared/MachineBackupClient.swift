import Foundation

/// SSE-compatible progress event (section/progress/log/error/done/download).
/// Polling maps job status into these so the progress window stays unchanged.
public struct BackupProgressEvent: Sendable {
    public var type: String
    public var message: String
    public var name: String
    public var status: String
    public var detail: String
    public var archiveToken: String

    public init(
        type: String,
        message: String = "",
        name: String = "",
        status: String = "",
        detail: String = "",
        archiveToken: String = ""
    ) {
        self.type = type
        self.message = message
        self.name = name
        self.status = status
        self.detail = detail
        self.archiveToken = archiveToken
    }

    public func formatLine() -> String {
        switch type {
        case "section":
            return BackupMenuFormatter.formatBackupProgressSection(message: message)
        case "progress":
            return BackupMenuFormatter.formatBackupProgressFrame(name: name, status: status, detail: detail)
        case "log":
            return BackupMenuFormatter.formatBackupProgressLog(message: message)
        case "error":
            return BackupMenuFormatter.formatBackupProgressError(message: message.isEmpty ? "backup stream error" : message)
        case "done":
            return BackupMenuFormatter.formatBackupProgressDone(message: message)
        default:
            return message
        }
    }
}

/// Daemon backup job snapshot (GET /backup/jobs/{id}).
public struct BackupJobStatus: Sendable {
    public var jobId: String
    public var status: String
    public var phase: String
    public var filesTotal: Int
    public var filesDone: Int
    public var archiveBytes: Int64
    public var archiveToken: String
    public var error: String

    public var isTerminal: Bool {
        status == "done" || status == "error" || status == "canceled"
    }
}

/// Downloads remote machine backups via short poll + archive_token GET.
///
/// Flow:
/// 1. POST `/api/remote-agent/machine/backup/jobs` with `{"archive":true,...}` (202 or 409 attach)
/// 2. GET `/api/remote-agent/machine/backup/jobs/{id}` until done/error
/// 3. GET `/api/remote-agent/machine/backup/archive?token=…`
public final class MachineBackupClient: @unchecked Sendable {
    public var baseURL: String
    public var token: String
    private let session: URLSession

    public init(baseURL: String = "", token: String = "", session: URLSession? = nil) {
        self.baseURL = ServiceClient.normalizeBaseURL(baseURL)
        self.token = token
        if let session {
            self.session = session
        } else {
            let cfg = URLSessionConfiguration.ephemeral
            cfg.timeoutIntervalForRequest = 30
            cfg.timeoutIntervalForResource = 6 * 60 * 60
            cfg.waitsForConnectivity = true
            self.session = URLSession(configuration: cfg)
        }
    }

    public func configure(baseURL: String, token: String) {
        self.baseURL = ServiceClient.normalizeBaseURL(baseURL)
        self.token = token
    }

    public var isConfigured: Bool { !baseURL.isEmpty }

    public static let backupJobsPath = "/api/remote-agent/machine/backup/jobs"
    public static let backupArchivePath = "/api/remote-agent/machine/backup/archive"

    public var pollIntervalNanoseconds: UInt64 = 2_000_000_000

    /// Start (or attach to) a job, poll until ready, download archive to destPath.
    @discardableResult
    public func downloadBackupArchive(
        to destPath: String,
        resumeJobID: String = "",
        onJobID: ((String) -> Void)? = nil,
        onProgress: ((BackupProgressEvent) -> Void)? = nil
    ) async throws -> Int64 {
        guard isConfigured else { throw ServiceClientError.notConfigured }
        var jobID = resumeJobID.trimmingCharacters(in: .whitespacesAndNewlines)
        if jobID.isEmpty {
            let started = try await startJob()
            jobID = started.jobId
        }
        if jobID.isEmpty {
            throw ServiceClientError.unreachable("backup jobs API returned no job_id; deploy the new ai-critic-server")
        }
        onJobID?(jobID)
        let done = try await waitUntilReady(jobID: jobID, onProgress: onProgress)
        if done.status != "done" {
            let msg = done.error.isEmpty ? done.status : done.error
            onProgress?(BackupProgressEvent(type: "error", message: msg))
            throw ServiceClientError.unreachable(msg)
        }
        let archiveToken = done.archiveToken
        if archiveToken.isEmpty {
            throw ServiceClientError.unreachable("backup job missing archive_token")
        }
        onProgress?(BackupProgressEvent(type: "done", message: "ready", archiveToken: archiveToken))
        onProgress?(BackupProgressEvent(type: "download", message: BackupMenuFormatter.formatBackupProgressDownloadStart()))
        var lastError: Error = ServiceClientError.unreachable("archive download failed")
        for _ in 0..<3 {
            do {
                return try await downloadArchiveByToken(archiveToken, to: destPath)
            } catch {
                lastError = error
                try await Task.sleep(nanoseconds: 1_000_000_000)
            }
        }
        throw lastError
    }

    public func startJob() async throws -> BackupJobStatus {
        guard isConfigured else { throw ServiceClientError.notConfigured }
        guard let url = URL(string: baseURL + Self.backupJobsPath) else {
            throw ServiceClientError.unreachable("invalid backup jobs url")
        }
        var request = URLRequest(url: url)
        request.httpMethod = "POST"
        request.setValue("application/json", forHTTPHeaderField: "Content-Type")
        request.timeoutInterval = 30
        applyAuth(&request)
        request.httpBody = try JSONSerialization.data(withJSONObject: [
            "archive": true,
            "exclude": [String](),
            "include": [String](),
        ])
        let (data, response) = try await session.data(for: request)
        guard let http = response as? HTTPURLResponse else {
            throw ServiceClientError.unreachable("backup job start failed")
        }
        let head = String(data: data.prefix(64), encoding: .utf8)?
            .trimmingCharacters(in: .whitespacesAndNewlines)
            .lowercased() ?? ""
        if head.hasPrefix("<!doctype") || head.hasPrefix("<html") {
            throw ServiceClientError.unreachable("backup jobs API not on this server (got HTML); deploy the new ai-critic-server")
        }
        if http.statusCode == 409 {
            if let job = parseWrappedJob(data), !job.jobId.isEmpty {
                return job
            }
            throw ServiceClientError.unreachable("backup job already running")
        }
        guard (200..<300).contains(http.statusCode) else {
            throw ServiceClientError.unreachable("backup job start failed (HTTP \(http.statusCode))")
        }
        let job = parseJob(data)
        if job.jobId.isEmpty {
            throw ServiceClientError.unreachable("backup jobs API returned no job_id (HTTP \(http.statusCode)); deploy the new ai-critic-server")
        }
        return job
    }

    public func getJob(id: String) async throws -> BackupJobStatus {
        let trimmed = id.trimmingCharacters(in: .whitespacesAndNewlines)
        guard !trimmed.isEmpty else {
            throw ServiceClientError.unreachable("job id is required")
        }
        guard let url = URL(string: baseURL + Self.backupJobsPath + "/" + trimmed) else {
            throw ServiceClientError.unreachable("invalid backup job url")
        }
        var request = URLRequest(url: url)
        request.httpMethod = "GET"
        request.timeoutInterval = 30
        applyAuth(&request)
        let (data, response) = try await session.data(for: request)
        guard let http = response as? HTTPURLResponse, (200..<300).contains(http.statusCode) else {
            throw ServiceClientError.unreachable("backup job poll failed")
        }
        return parseJob(data)
    }

    public func waitUntilReady(jobID: String, onProgress: ((BackupProgressEvent) -> Void)?) async throws -> BackupJobStatus {
        var lastLine = ""
        while !Task.isCancelled {
            let job = try await getJob(id: jobID)
            let line = "\(job.status) \(job.phase) \(job.filesDone)/\(job.filesTotal)"
            if line != lastLine {
                lastLine = line
                onProgress?(BackupProgressEvent(type: "section", message: job.phase.isEmpty ? job.status : job.phase))
                onProgress?(BackupProgressEvent(
                    type: "progress",
                    name: job.jobId,
                    status: job.status,
                    detail: "files=\(job.filesDone)/\(job.filesTotal)"
                ))
            }
            if job.isTerminal {
                return job
            }
            try await Task.sleep(nanoseconds: pollIntervalNanoseconds)
        }
        throw ServiceClientError.unreachable("backup job cancelled")
    }

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
        request.timeoutInterval = 300
        applyAuth(&request)

        let (tempURL, response) = try await session.download(for: request)
        guard let http = response as? HTTPURLResponse, (200..<300).contains(http.statusCode) else {
            throw ServiceClientError.unreachable("archive download failed")
        }

        let dest = URL(fileURLWithPath: destPath)
        let dir = dest.deletingLastPathComponent()
        try FileManager.default.createDirectory(at: dir, withIntermediateDirectories: true)
        if FileManager.default.fileExists(atPath: destPath) {
            try FileManager.default.removeItem(at: dest)
        }
        try FileManager.default.moveItem(at: tempURL, to: dest)
        let attrs = try FileManager.default.attributesOfItem(atPath: destPath)
        return (attrs[.size] as? NSNumber)?.int64Value ?? 0
    }

    private func parseWrappedJob(_ data: Data) -> BackupJobStatus? {
        guard let obj = try? JSONSerialization.jsonObject(with: data) as? [String: Any] else {
            return nil
        }
        if let job = obj["job"] as? [String: Any] {
            return parseJobDict(job)
        }
        return nil
    }

    private func parseJob(_ data: Data) -> BackupJobStatus {
        let obj = (try? JSONSerialization.jsonObject(with: data) as? [String: Any]) ?? [:]
        return parseJobDict(obj)
    }

    private func parseJobDict(_ obj: [String: Any]) -> BackupJobStatus {
        func str(_ key: String) -> String {
            if let s = obj[key] as? String { return s }
            return ""
        }
        func intVal(_ key: String) -> Int {
            if let n = obj[key] as? Int { return n }
            if let n = obj[key] as? Int64 { return Int(n) }
            if let n = obj[key] as? Double { return Int(n) }
            return 0
        }
        func int64Val(_ key: String) -> Int64 {
            if let n = obj[key] as? Int64 { return n }
            if let n = obj[key] as? Int { return Int64(n) }
            if let n = obj[key] as? Double { return Int64(n) }
            return 0
        }
        return BackupJobStatus(
            jobId: str("job_id"),
            status: str("status"),
            phase: str("phase"),
            filesTotal: intVal("files_total"),
            filesDone: intVal("files_done"),
            archiveBytes: int64Val("archive_bytes"),
            archiveToken: str("archive_token"),
            error: str("error")
        )
    }

    private func applyAuth(_ request: inout URLRequest) {
        let header = ServiceClient.authorizationHeader(token: token)
        if !header.isEmpty {
            request.setValue(header, forHTTPHeaderField: "Authorization")
        }
    }
}
