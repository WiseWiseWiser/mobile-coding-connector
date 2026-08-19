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
