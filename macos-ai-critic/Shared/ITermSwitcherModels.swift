import Foundation

/// GET /api/local/iterm2/inventory
public struct ITermInventory: Decodable, Equatable {
    public let itermRunning: Bool
    public let desktops: [ITermDesktopGroup]
    public let savedNotes: [ITermOrphanNote]
    public let cachedAt: String
    public let fromCache: Bool
    public let refreshing: Bool

    enum CodingKeys: String, CodingKey {
        case itermRunning = "iterm_running"
        case desktops
        case savedNotes = "saved_notes"
        case cachedAt = "cached_at"
        case fromCache = "from_cache"
        case refreshing
    }

    public init(
        itermRunning: Bool = false,
        desktops: [ITermDesktopGroup] = [],
        savedNotes: [ITermOrphanNote] = [],
        cachedAt: String = "",
        fromCache: Bool = false,
        refreshing: Bool = false
    ) {
        self.itermRunning = itermRunning
        self.desktops = desktops
        self.savedNotes = savedNotes
        self.cachedAt = cachedAt
        self.fromCache = fromCache
        self.refreshing = refreshing
    }

    public init(from decoder: Decoder) throws {
        let c = try decoder.container(keyedBy: CodingKeys.self)
        itermRunning = try c.decodeIfPresent(Bool.self, forKey: .itermRunning) ?? false
        desktops = try c.decodeIfPresent([ITermDesktopGroup].self, forKey: .desktops) ?? []
        savedNotes = try c.decodeIfPresent([ITermOrphanNote].self, forKey: .savedNotes) ?? []
        cachedAt = try c.decodeIfPresent(String.self, forKey: .cachedAt) ?? ""
        fromCache = try c.decodeIfPresent(Bool.self, forKey: .fromCache) ?? false
        refreshing = try c.decodeIfPresent(Bool.self, forKey: .refreshing) ?? false
    }
}

/// One SSE frame from GET /api/local/iterm2/inventory/stream.
public struct ITermInventoryStreamFrame: Decodable {
    public let type: String
    public let inventory: ITermInventory?
    public let message: String?
}

public struct ITermDesktopGroup: Decodable, Equatable, Identifiable {
    public var id: Int { spaceIndex }
    public let spaceIndex: Int
    public let desktop: Int
    public let spaceID: UInt64
    public let spaceUUID: String
    public let label: String
    public let current: Bool
    public let sessions: [ITermLiveSession]

    enum CodingKeys: String, CodingKey {
        case spaceIndex = "space_index"
        case desktop
        case spaceID = "space_id"
        case spaceUUID = "space_uuid"
        case label
        case current
        case sessions
    }

    public init(
        spaceIndex: Int,
        desktop: Int,
        spaceID: UInt64 = 0,
        spaceUUID: String = "",
        label: String = "",
        current: Bool = false,
        sessions: [ITermLiveSession] = []
    ) {
        self.spaceIndex = spaceIndex
        self.desktop = desktop
        self.spaceID = spaceID
        self.spaceUUID = spaceUUID
        self.label = label
        self.current = current
        self.sessions = sessions
    }

    public init(from decoder: Decoder) throws {
        let c = try decoder.container(keyedBy: CodingKeys.self)
        spaceIndex = try c.decodeIfPresent(Int.self, forKey: .spaceIndex) ?? 0
        desktop = try c.decodeIfPresent(Int.self, forKey: .desktop) ?? (spaceIndex + 1)
        spaceID = try c.decodeIfPresent(UInt64.self, forKey: .spaceID) ?? 0
        spaceUUID = try c.decodeIfPresent(String.self, forKey: .spaceUUID) ?? ""
        label = try c.decodeIfPresent(String.self, forKey: .label) ?? ""
        current = try c.decodeIfPresent(Bool.self, forKey: .current) ?? false
        sessions = try c.decodeIfPresent([ITermLiveSession].self, forKey: .sessions) ?? []
    }
}

public struct ITermLiveSession: Decodable, Equatable, Identifiable {
    public var id: String { sessionID }
    public let sessionID: String
    public let sessionName: String
    public let windowID: String
    public let windowName: String
    public let tabIndex: Int
    public let tabName: String
    public let cwd: String
    public let idle: Bool?
    public let note: String
    public let bookmarked: Bool
    public let spaceIndex: Int
    public let desktop: Int
    public let agentRunner: String
    public let grokSessionID: String
    public let pid: Int
    public let mark: String

    enum CodingKeys: String, CodingKey {
        case sessionID = "session_id"
        case sessionName = "session_name"
        case windowID = "window_id"
        case windowName = "window_name"
        case tabIndex = "tab_index"
        case tabName = "tab_name"
        case cwd
        case idle
        case note
        case bookmarked
        case spaceIndex = "space_index"
        case desktop
        case agentRunner = "agent_runner"
        case grokSessionID = "grok_session_id"
        case pid
        case mark
    }

    public init(
        sessionID: String,
        sessionName: String = "",
        windowID: String = "",
        windowName: String = "",
        tabIndex: Int = 0,
        tabName: String = "",
        cwd: String = "",
        idle: Bool? = nil,
        note: String = "",
        bookmarked: Bool = false,
        spaceIndex: Int = 0,
        desktop: Int = 1,
        agentRunner: String = "",
        grokSessionID: String = "",
        pid: Int = 0,
        mark: String = ""
    ) {
        self.sessionID = sessionID
        self.sessionName = sessionName
        self.windowID = windowID
        self.windowName = windowName
        self.tabIndex = tabIndex
        self.tabName = tabName
        self.cwd = cwd
        self.idle = idle
        self.note = note
        self.bookmarked = bookmarked
        self.spaceIndex = spaceIndex
        self.desktop = desktop
        self.agentRunner = agentRunner
        self.grokSessionID = grokSessionID
        self.pid = pid
        self.mark = mark
    }

    public init(from decoder: Decoder) throws {
        let c = try decoder.container(keyedBy: CodingKeys.self)
        sessionID = try c.decodeIfPresent(String.self, forKey: .sessionID) ?? ""
        sessionName = try c.decodeIfPresent(String.self, forKey: .sessionName) ?? ""
        windowID = try c.decodeIfPresent(String.self, forKey: .windowID) ?? ""
        windowName = try c.decodeIfPresent(String.self, forKey: .windowName) ?? ""
        tabIndex = try c.decodeIfPresent(Int.self, forKey: .tabIndex) ?? 0
        tabName = try c.decodeIfPresent(String.self, forKey: .tabName) ?? ""
        cwd = try c.decodeIfPresent(String.self, forKey: .cwd) ?? ""
        idle = try c.decodeIfPresent(Bool.self, forKey: .idle)
        note = try c.decodeIfPresent(String.self, forKey: .note) ?? ""
        bookmarked = try c.decodeIfPresent(Bool.self, forKey: .bookmarked) ?? false
        spaceIndex = try c.decodeIfPresent(Int.self, forKey: .spaceIndex) ?? 0
        desktop = try c.decodeIfPresent(Int.self, forKey: .desktop) ?? (spaceIndex + 1)
        agentRunner = try c.decodeIfPresent(String.self, forKey: .agentRunner) ?? ""
        grokSessionID = try c.decodeIfPresent(String.self, forKey: .grokSessionID) ?? ""
        pid = try c.decodeIfPresent(Int.self, forKey: .pid) ?? 0
        mark = try c.decodeIfPresent(String.self, forKey: .mark) ?? ""
    }
}

public struct ITermOrphanNote: Decodable, Equatable, Identifiable {
    public var id: String { sessionID }
    public let sessionID: String
    public let note: String
    public let bookmarked: Bool
    public let sessionName: String
    public let windowName: String
    public let cwd: String
    public let spaceIndex: Int
    public let updatedAt: String

    enum CodingKeys: String, CodingKey {
        case sessionID = "session_id"
        case note
        case bookmarked
        case sessionName = "session_name"
        case windowName = "window_name"
        case cwd
        case spaceIndex = "space_index"
        case updatedAt = "updated_at"
    }

    public init(
        sessionID: String,
        note: String,
        bookmarked: Bool = false,
        sessionName: String = "",
        windowName: String = "",
        cwd: String = "",
        spaceIndex: Int = 0,
        updatedAt: String = ""
    ) {
        self.sessionID = sessionID
        self.note = note
        self.bookmarked = bookmarked
        self.sessionName = sessionName
        self.windowName = windowName
        self.cwd = cwd
        self.spaceIndex = spaceIndex
        self.updatedAt = updatedAt
    }

    public init(from decoder: Decoder) throws {
        let c = try decoder.container(keyedBy: CodingKeys.self)
        sessionID = try c.decodeIfPresent(String.self, forKey: .sessionID) ?? ""
        note = try c.decodeIfPresent(String.self, forKey: .note) ?? ""
        bookmarked = try c.decodeIfPresent(Bool.self, forKey: .bookmarked) ?? false
        sessionName = try c.decodeIfPresent(String.self, forKey: .sessionName) ?? ""
        windowName = try c.decodeIfPresent(String.self, forKey: .windowName) ?? ""
        cwd = try c.decodeIfPresent(String.self, forKey: .cwd) ?? ""
        spaceIndex = try c.decodeIfPresent(Int.self, forKey: .spaceIndex) ?? 0
        updatedAt = try c.decodeIfPresent(String.self, forKey: .updatedAt) ?? ""
    }
}

public enum ITermSwitcherHotKey {
    /// kVK_Space
    public static let defaultKeyCode = 49
    /// cmdKey | shiftKey (Carbon)
    public static let defaultModifiers = 256 | 512
    public static let keyCodeDefaultsKey = "itermSwitcherKeyCode"
    public static let modifiersDefaultsKey = "itermSwitcherModifiers"
}
