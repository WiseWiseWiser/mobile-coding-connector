import Foundation

/// Server terminal session — mirrors Go `client.TerminalSession` / GET /api/terminal/sessions.
public struct TerminalSession: Decodable, Identifiable, Equatable {
    public let id: String
    public let name: String
    public let cwd: String
    public let createdAt: String
    public let status: String
    public let connected: Bool

    enum CodingKeys: String, CodingKey {
        case id
        case name
        case cwd
        case createdAt = "created_at"
        case status
        case connected
    }

    public init(
        id: String,
        name: String = "",
        cwd: String = "",
        createdAt: String = "",
        status: String = "",
        connected: Bool = false
    ) {
        self.id = id
        self.name = name
        self.cwd = cwd
        self.createdAt = createdAt
        self.status = status
        self.connected = connected
    }

    public init(from decoder: Decoder) throws {
        let c = try decoder.container(keyedBy: CodingKeys.self)
        id = try c.decodeIfPresent(String.self, forKey: .id) ?? ""
        name = try c.decodeIfPresent(String.self, forKey: .name) ?? ""
        cwd = try c.decodeIfPresent(String.self, forKey: .cwd) ?? ""
        createdAt = try c.decodeIfPresent(String.self, forKey: .createdAt) ?? ""
        status = try c.decodeIfPresent(String.self, forKey: .status) ?? ""
        connected = try c.decodeIfPresent(Bool.self, forKey: .connected) ?? false
    }
}

/// Paginated list response from GET /api/terminal/sessions.
public struct TerminalSessionsPage: Decodable {
    public let sessions: [TerminalSession]
    public let page: Int
    public let pageSize: Int
    public let total: Int
    public let totalPages: Int

    enum CodingKeys: String, CodingKey {
        case sessions
        case page
        case pageSize = "page_size"
        case total
        case totalPages = "total_pages"
    }

    public init(
        sessions: [TerminalSession] = [],
        page: Int = 1,
        pageSize: Int = 100,
        total: Int = 0,
        totalPages: Int = 0
    ) {
        self.sessions = sessions
        self.page = page
        self.pageSize = pageSize
        self.total = total
        self.totalPages = totalPages
    }

    public init(from decoder: Decoder) throws {
        let c = try decoder.container(keyedBy: CodingKeys.self)
        sessions = try c.decodeIfPresent([TerminalSession].self, forKey: .sessions) ?? []
        page = try c.decodeIfPresent(Int.self, forKey: .page) ?? 1
        pageSize = try c.decodeIfPresent(Int.self, forKey: .pageSize) ?? 100
        total = try c.decodeIfPresent(Int.self, forKey: .total) ?? sessions.count
        totalPages = try c.decodeIfPresent(Int.self, forKey: .totalPages) ?? 1
    }
}
