import Foundation

/// GET /api/local/skills
public struct SkillsListResponse: Decodable, Equatable {
    public let skills: [SkillsPickerItem]
    public let missingRoots: [String]

    enum CodingKeys: String, CodingKey {
        case skills
        case missingRoots = "missing_roots"
    }

    public init(skills: [SkillsPickerItem] = [], missingRoots: [String] = []) {
        self.skills = skills
        self.missingRoots = missingRoots
    }

    public init(from decoder: Decoder) throws {
        let c = try decoder.container(keyedBy: CodingKeys.self)
        skills = try c.decodeIfPresent([SkillsPickerItem].self, forKey: .skills) ?? []
        missingRoots = try c.decodeIfPresent([String].self, forKey: .missingRoots) ?? []
    }
}

/// POST /api/local/skills/use
public struct SkillsUseResponse: Decodable {
    public let skill: SkillsPickerItem
}

public struct FuzzySpan: Decodable, Equatable {
    public let text: String
    public let matched: Bool

    enum CodingKeys: String, CodingKey {
        case text, matched
    }

    public init(text: String, matched: Bool) {
        self.text = text
        self.matched = matched
    }

    public init(from decoder: Decoder) throws {
        let c = try decoder.container(keyedBy: CodingKeys.self)
        text = try c.decodeIfPresent(String.self, forKey: .text) ?? ""
        matched = try c.decodeIfPresent(Bool.self, forKey: .matched) ?? false
    }
}

public struct SkillsPickerItem: Decodable, Equatable, Identifiable {
    public var id: String { path }
    public let name: String
    public let fmName: String
    public let dir: String
    public let path: String
    public let useCount: Int
    public let lastUsed: String
    public let titleSpans: [FuzzySpan]
    public let pathSpans: [FuzzySpan]

    enum CodingKeys: String, CodingKey {
        case name
        case fmName = "fm_name"
        case dir
        case path
        case useCount = "use_count"
        case lastUsed = "last_used"
        case titleSpans = "title_spans"
        case pathSpans = "path_spans"
    }

    public init(
        name: String,
        fmName: String = "",
        dir: String,
        path: String,
        useCount: Int = 0,
        lastUsed: String = "",
        titleSpans: [FuzzySpan] = [],
        pathSpans: [FuzzySpan] = []
    ) {
        self.name = name
        self.fmName = fmName
        self.dir = dir
        self.path = path
        self.useCount = useCount
        self.lastUsed = lastUsed
        self.titleSpans = titleSpans
        self.pathSpans = pathSpans
    }

    public init(from decoder: Decoder) throws {
        let c = try decoder.container(keyedBy: CodingKeys.self)
        name = try c.decodeIfPresent(String.self, forKey: .name) ?? ""
        fmName = try c.decodeIfPresent(String.self, forKey: .fmName) ?? ""
        dir = try c.decodeIfPresent(String.self, forKey: .dir) ?? ""
        path = try c.decodeIfPresent(String.self, forKey: .path) ?? ""
        useCount = try c.decodeIfPresent(Int.self, forKey: .useCount) ?? 0
        lastUsed = try c.decodeIfPresent(String.self, forKey: .lastUsed) ?? ""
        titleSpans = try c.decodeIfPresent([FuzzySpan].self, forKey: .titleSpans) ?? []
        pathSpans = try c.decodeIfPresent([FuzzySpan].self, forKey: .pathSpans) ?? []
    }
}

public enum SkillsPickerHotKey {
    /// kVK_ANSI_Semicolon
    public static let defaultKeyCode = 41
    /// cmdKey | shiftKey (Carbon)
    public static let defaultModifiers = 256 | 512
}

/// GET /api/local/clipboard/peek
public struct ClipboardPeekResponse: Decodable, Equatable {
    public let kind: String
    public let ext: String
    public let bytes: Int
    public let preview: String
    public let mime: String
    public let available: [String]

    enum CodingKeys: String, CodingKey {
        case kind, ext, bytes, preview, mime, available
    }

    public init(
        kind: String = "empty",
        ext: String = "",
        bytes: Int = 0,
        preview: String = "",
        mime: String = "",
        available: [String] = []
    ) {
        self.kind = kind
        self.ext = ext
        self.bytes = bytes
        self.preview = preview
        self.mime = mime
        self.available = available
    }

    public init(from decoder: Decoder) throws {
        let c = try decoder.container(keyedBy: CodingKeys.self)
        kind = try c.decodeIfPresent(String.self, forKey: .kind) ?? "empty"
        ext = try c.decodeIfPresent(String.self, forKey: .ext) ?? ""
        bytes = try c.decodeIfPresent(Int.self, forKey: .bytes) ?? 0
        preview = try c.decodeIfPresent(String.self, forKey: .preview) ?? ""
        mime = try c.decodeIfPresent(String.self, forKey: .mime) ?? ""
        available = try c.decodeIfPresent([String].self, forKey: .available) ?? []
    }
}

/// POST /api/local/clipboard/dump
public struct ClipboardDumpResponse: Decodable, Equatable {
    public let path: String
    public let kind: String
    public let ext: String
    public let bytes: Int

    enum CodingKeys: String, CodingKey {
        case path, kind, ext, bytes
    }

    public init(path: String, kind: String = "", ext: String = "", bytes: Int = 0) {
        self.path = path
        self.kind = kind
        self.ext = ext
        self.bytes = bytes
    }

    public init(from decoder: Decoder) throws {
        let c = try decoder.container(keyedBy: CodingKeys.self)
        path = try c.decodeIfPresent(String.self, forKey: .path) ?? ""
        kind = try c.decodeIfPresent(String.self, forKey: .kind) ?? ""
        ext = try c.decodeIfPresent(String.self, forKey: .ext) ?? ""
        bytes = try c.decodeIfPresent(Int.self, forKey: .bytes) ?? 0
    }
}

/// GET/PUT /api/local/adhoc
public struct AdhocTextResponse: Decodable, Equatable {
    public let content: String
    public let path: String

    enum CodingKeys: String, CodingKey {
        case content, path
    }

    public init(content: String = "", path: String = "") {
        self.content = content
        self.path = path
    }

    public init(from decoder: Decoder) throws {
        let c = try decoder.container(keyedBy: CodingKeys.self)
        content = try c.decodeIfPresent(String.self, forKey: .content) ?? ""
        path = try c.decodeIfPresent(String.self, forKey: .path) ?? ""
    }
}

/// GET /api/local/templates
public struct TemplatesListResponse: Decodable, Equatable {
    public let templates: [TemplatesPickerItem]
    public let missingRoots: [String]
    public let roots: [TemplatesRootItem]

    enum CodingKeys: String, CodingKey {
        case templates
        case missingRoots = "missing_roots"
        case roots
    }

    public init(
        templates: [TemplatesPickerItem] = [],
        missingRoots: [String] = [],
        roots: [TemplatesRootItem] = []
    ) {
        self.templates = templates
        self.missingRoots = missingRoots
        self.roots = roots
    }

    public init(from decoder: Decoder) throws {
        let c = try decoder.container(keyedBy: CodingKeys.self)
        templates = try c.decodeIfPresent([TemplatesPickerItem].self, forKey: .templates) ?? []
        missingRoots = try c.decodeIfPresent([String].self, forKey: .missingRoots) ?? []
        roots = try c.decodeIfPresent([TemplatesRootItem].self, forKey: .roots) ?? []
    }
}

/// One registered template root from list / add-dir.
public struct TemplatesRootItem: Decodable, Equatable, Identifiable {
    public var id: String { path }
    public let path: String
    public let note: String

    enum CodingKeys: String, CodingKey {
        case path, note
    }

    public init(path: String, note: String = "") {
        self.path = path
        self.note = note
    }

    public init(from decoder: Decoder) throws {
        let c = try decoder.container(keyedBy: CodingKeys.self)
        path = try c.decodeIfPresent(String.self, forKey: .path) ?? ""
        note = try c.decodeIfPresent(String.self, forKey: .note) ?? ""
    }
}

/// POST /api/local/templates/add-dir
public struct TemplatesAddDirResponse: Decodable, Equatable {
    public let root: TemplatesRootItem
    public let duplicate: Bool

    enum CodingKeys: String, CodingKey {
        case root, duplicate
    }

    public init(root: TemplatesRootItem, duplicate: Bool = false) {
        self.root = root
        self.duplicate = duplicate
    }

    public init(from decoder: Decoder) throws {
        let c = try decoder.container(keyedBy: CodingKeys.self)
        root = try c.decodeIfPresent(TemplatesRootItem.self, forKey: .root) ?? TemplatesRootItem(path: "")
        duplicate = try c.decodeIfPresent(Bool.self, forKey: .duplicate) ?? false
    }
}

/// POST /api/local/templates/create
public struct TemplatesCreateResponse: Decodable, Equatable {
    public let template: TemplatesPickerItem

    public init(template: TemplatesPickerItem) {
        self.template = template
    }
}

/// POST /api/local/files/add
public struct FilesAddResponse: Decodable, Equatable {
    public let file: FilesPickerItem
    public let duplicate: Bool

    enum CodingKeys: String, CodingKey {
        case file, duplicate
    }

    public init(file: FilesPickerItem, duplicate: Bool = false) {
        self.file = file
        self.duplicate = duplicate
    }

    public init(from decoder: Decoder) throws {
        let c = try decoder.container(keyedBy: CodingKeys.self)
        file = try c.decodeIfPresent(FilesPickerItem.self, forKey: .file) ?? FilesPickerItem(path: "")
        duplicate = try c.decodeIfPresent(Bool.self, forKey: .duplicate) ?? false
    }
}

public struct TemplatesPickerItem: Decodable, Equatable, Identifiable {
    public var id: String { path }
    public let name: String
    public let fmName: String
    public let description: String
    public let tags: [String]
    public let path: String
    public let body: String
    public let useCount: Int
    public let lastUsed: String
    public let score: Int
    public let titleSpans: [FuzzySpan]
    public let pathSpans: [FuzzySpan]
    public let bodySpans: [FuzzySpan]

    enum CodingKeys: String, CodingKey {
        case name
        case fmName = "fm_name"
        case description
        case tags
        case path
        case body
        case useCount = "use_count"
        case lastUsed = "last_used"
        case score
        case titleSpans = "title_spans"
        case pathSpans = "path_spans"
        case bodySpans = "body_spans"
    }

    public init(
        name: String,
        fmName: String = "",
        description: String = "",
        tags: [String] = [],
        path: String,
        body: String = "",
        useCount: Int = 0,
        lastUsed: String = "",
        score: Int = 0,
        titleSpans: [FuzzySpan] = [],
        pathSpans: [FuzzySpan] = [],
        bodySpans: [FuzzySpan] = []
    ) {
        self.name = name
        self.fmName = fmName
        self.description = description
        self.tags = tags
        self.path = path
        self.body = body
        self.useCount = useCount
        self.lastUsed = lastUsed
        self.score = score
        self.titleSpans = titleSpans
        self.pathSpans = pathSpans
        self.bodySpans = bodySpans
    }

    public init(from decoder: Decoder) throws {
        let c = try decoder.container(keyedBy: CodingKeys.self)
        name = try c.decodeIfPresent(String.self, forKey: .name) ?? ""
        fmName = try c.decodeIfPresent(String.self, forKey: .fmName) ?? ""
        description = try c.decodeIfPresent(String.self, forKey: .description) ?? ""
        tags = try c.decodeIfPresent([String].self, forKey: .tags) ?? []
        path = try c.decodeIfPresent(String.self, forKey: .path) ?? ""
        body = try c.decodeIfPresent(String.self, forKey: .body) ?? ""
        useCount = try c.decodeIfPresent(Int.self, forKey: .useCount) ?? 0
        lastUsed = try c.decodeIfPresent(String.self, forKey: .lastUsed) ?? ""
        score = try c.decodeIfPresent(Int.self, forKey: .score) ?? 0
        titleSpans = try c.decodeIfPresent([FuzzySpan].self, forKey: .titleSpans) ?? []
        pathSpans = try c.decodeIfPresent([FuzzySpan].self, forKey: .pathSpans) ?? []
        bodySpans = try c.decodeIfPresent([FuzzySpan].self, forKey: .bodySpans) ?? []
    }
}

/// GET /api/local/files
public struct FilesListResponse: Decodable, Equatable {
    public let files: [FilesPickerItem]

    enum CodingKeys: String, CodingKey {
        case files
    }

    public init(files: [FilesPickerItem] = []) {
        self.files = files
    }

    public init(from decoder: Decoder) throws {
        let c = try decoder.container(keyedBy: CodingKeys.self)
        files = try c.decodeIfPresent([FilesPickerItem].self, forKey: .files) ?? []
    }
}

public struct FilesPickerItem: Decodable, Equatable, Identifiable {
    public var id: String { path }
    public let path: String
    public let name: String
    public let note: String
    public let exists: Bool
    public let isDir: Bool
    public let useCount: Int
    public let lastUsed: String
    public let score: Int
    public let titleSpans: [FuzzySpan]
    public let pathSpans: [FuzzySpan]

    enum CodingKeys: String, CodingKey {
        case path
        case name
        case note
        case exists
        case isDir = "is_dir"
        case useCount = "use_count"
        case lastUsed = "last_used"
        case score
        case titleSpans = "title_spans"
        case pathSpans = "path_spans"
    }

    public init(
        path: String,
        name: String = "",
        note: String = "",
        exists: Bool = false,
        isDir: Bool = false,
        useCount: Int = 0,
        lastUsed: String = "",
        score: Int = 0,
        titleSpans: [FuzzySpan] = [],
        pathSpans: [FuzzySpan] = []
    ) {
        self.path = path
        self.name = name
        self.note = note
        self.exists = exists
        self.isDir = isDir
        self.useCount = useCount
        self.lastUsed = lastUsed
        self.score = score
        self.titleSpans = titleSpans
        self.pathSpans = pathSpans
    }

    public init(from decoder: Decoder) throws {
        let c = try decoder.container(keyedBy: CodingKeys.self)
        path = try c.decodeIfPresent(String.self, forKey: .path) ?? ""
        name = try c.decodeIfPresent(String.self, forKey: .name) ?? ""
        note = try c.decodeIfPresent(String.self, forKey: .note) ?? ""
        exists = try c.decodeIfPresent(Bool.self, forKey: .exists) ?? false
        isDir = try c.decodeIfPresent(Bool.self, forKey: .isDir) ?? false
        useCount = try c.decodeIfPresent(Int.self, forKey: .useCount) ?? 0
        lastUsed = try c.decodeIfPresent(String.self, forKey: .lastUsed) ?? ""
        score = try c.decodeIfPresent(Int.self, forKey: .score) ?? 0
        titleSpans = try c.decodeIfPresent([FuzzySpan].self, forKey: .titleSpans) ?? []
        pathSpans = try c.decodeIfPresent([FuzzySpan].self, forKey: .pathSpans) ?? []
    }
}

/// POST /api/local/commands/add
public struct CommandsAddResponse: Decodable, Equatable {
    public let command: CommandsPickerItem
    public let duplicate: Bool

    enum CodingKeys: String, CodingKey {
        case command, duplicate
    }

    public init(command: CommandsPickerItem, duplicate: Bool = false) {
        self.command = command
        self.duplicate = duplicate
    }

    public init(from decoder: Decoder) throws {
        let c = try decoder.container(keyedBy: CodingKeys.self)
        command = try c.decodeIfPresent(CommandsPickerItem.self, forKey: .command) ?? CommandsPickerItem(command: "")
        duplicate = try c.decodeIfPresent(Bool.self, forKey: .duplicate) ?? false
    }
}

/// GET /api/local/commands
public struct CommandsListResponse: Decodable, Equatable {
    public let commands: [CommandsPickerItem]

    enum CodingKeys: String, CodingKey {
        case commands
    }

    public init(commands: [CommandsPickerItem] = []) {
        self.commands = commands
    }

    public init(from decoder: Decoder) throws {
        let c = try decoder.container(keyedBy: CodingKeys.self)
        commands = try c.decodeIfPresent([CommandsPickerItem].self, forKey: .commands) ?? []
    }
}

public struct CommandsPickerItem: Decodable, Equatable, Identifiable {
    public var id: String { command }
    public let command: String
    public let name: String
    public let note: String
    public let useCount: Int
    public let lastUsed: String
    public let score: Int
    public let titleSpans: [FuzzySpan]
    public let commandSpans: [FuzzySpan]

    enum CodingKeys: String, CodingKey {
        case command
        case name
        case note
        case useCount = "use_count"
        case lastUsed = "last_used"
        case score
        case titleSpans = "title_spans"
        case commandSpans = "command_spans"
    }

    public init(
        command: String,
        name: String = "",
        note: String = "",
        useCount: Int = 0,
        lastUsed: String = "",
        score: Int = 0,
        titleSpans: [FuzzySpan] = [],
        commandSpans: [FuzzySpan] = []
    ) {
        self.command = command
        self.name = name
        self.note = note
        self.useCount = useCount
        self.lastUsed = lastUsed
        self.score = score
        self.titleSpans = titleSpans
        self.commandSpans = commandSpans
    }

    public init(from decoder: Decoder) throws {
        let c = try decoder.container(keyedBy: CodingKeys.self)
        command = try c.decodeIfPresent(String.self, forKey: .command) ?? ""
        name = try c.decodeIfPresent(String.self, forKey: .name) ?? ""
        note = try c.decodeIfPresent(String.self, forKey: .note) ?? ""
        useCount = try c.decodeIfPresent(Int.self, forKey: .useCount) ?? 0
        lastUsed = try c.decodeIfPresent(String.self, forKey: .lastUsed) ?? ""
        score = try c.decodeIfPresent(Int.self, forKey: .score) ?? 0
        titleSpans = try c.decodeIfPresent([FuzzySpan].self, forKey: .titleSpans) ?? []
        commandSpans = try c.decodeIfPresent([FuzzySpan].self, forKey: .commandSpans) ?? []
    }
}
