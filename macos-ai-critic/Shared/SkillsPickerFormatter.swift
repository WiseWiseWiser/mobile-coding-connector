import Foundation

/// Pure formatters for the ⌘⇧; insert picker (skills + templates + files).
public enum SkillsPickerFormatter {
    public static let sidebarAll = "all"
    public static let sidebarSkills = "skills"
    public static let sidebarTemplates = "templates"
    public static let sidebarFiles = "files"
    public static let sidebarDefaultsKey = "insertPickerSidebarID"
    public static let sidebarWidthDefaultsKey = "insertPickerSidebarWidth"

    /// Default / collapsed icon-rail width (points).
    public static let sidebarIconWidth: Double = 52
    /// Titles appear when sidebar width is at least this (points).
    public static let sidebarTitleRevealWidth: Double = 72
    public static let sidebarMinWidth: Double = 44
    public static let sidebarMaxWidth: Double = 180
    /// Title fade duration when crossing the reveal threshold (seconds).
    public static let sidebarTitleAnimationSeconds: Double = 0.18

    public static func shouldShowSidebarTitles(width: Double) -> Bool {
        width >= sidebarTitleRevealWidth
    }

    /// Clamp and sanitize a persisted/dragged sidebar width.
    public static func clampSidebarWidth(_ width: Double) -> Double {
        if width.isNaN || width.isInfinite || width <= 0 {
            return sidebarIconWidth
        }
        return min(sidebarMaxWidth, max(sidebarMinWidth, width))
    }

    public static func formatSidebarSymbol(id: String) -> String {
        switch normalizeSidebarID(id) {
        case sidebarSkills: return "wrench.and.screwdriver"
        case sidebarTemplates: return "doc.text"
        case sidebarFiles: return "folder"
        default: return "square.grid.2x2"
        }
    }

    public static func formatWindowTitle() -> String {
        "Insert"
    }

    public static func formatHotKey() -> String {
        "⌘⇧;"
    }

    public static func formatCopiedToast() -> String {
        "Copied"
    }

    public static func formatSidebarTitle(id: String) -> String {
        switch id.trimmingCharacters(in: .whitespacesAndNewlines) {
        case "", sidebarAll: return "All"
        case sidebarSkills: return "Skills"
        case sidebarTemplates: return "Templates"
        case sidebarFiles: return "Files"
        default: return ""
        }
    }

    public static func normalizeSidebarID(_ id: String?) -> String {
        let trimmed = (id ?? "").trimmingCharacters(in: .whitespacesAndNewlines)
        switch trimmed {
        case sidebarAll, sidebarSkills, sidebarTemplates, sidebarFiles:
            return trimmed
        default:
            return sidebarAll
        }
    }

    public static func formatKindBadge(_ kind: InsertPickerKind) -> String {
        switch kind {
        case .skill: return "SKILL"
        case .template: return "TEMPLATE"
        case .file: return "FILE"
        }
    }

    /// Next list row for ↑/↓. `current` is nil when nothing is selected.
    /// - Down from nil → first (0); up from nil → last (count-1).
    /// - Clamps at ends; empty list → nil.
    public static func nextListSelectionIndex(count: Int, current: Int?, delta: Int) -> Int? {
        guard count > 0, delta != 0 else { return nil }
        let seed: Int
        if let current, current >= 0, current < count {
            seed = current
        } else if delta > 0 {
            seed = -1
        } else {
            seed = count
        }
        var next = seed + delta
        if next < 0 { next = 0 }
        if next >= count { next = count - 1 }
        return next
    }

    public static func formatTitle(_ skill: SkillsPickerItem) -> String {
        let fm = skill.fmName.trimmingCharacters(in: .whitespacesAndNewlines)
        if !fm.isEmpty { return fm }
        let name = skill.name.trimmingCharacters(in: .whitespacesAndNewlines)
        if !name.isEmpty { return name }
        return URL(fileURLWithPath: skill.path).deletingLastPathComponent().lastPathComponent
    }

    public static func formatSubtitle(_ skill: SkillsPickerItem) -> String {
        skill.path
    }

    /// Max characters for template body preview (second list line).
    public static let templateBodyPreviewMaxChars: Int = 96

    public static func formatTemplateTitle(_ template: TemplatesPickerItem) -> String {
        let fm = template.fmName.trimmingCharacters(in: .whitespacesAndNewlines)
        if !fm.isEmpty { return fm }
        let name = template.name.trimmingCharacters(in: .whitespacesAndNewlines)
        if !name.isEmpty { return name }
        return URL(fileURLWithPath: template.path).deletingPathExtension().lastPathComponent
    }

    /// Trailing description on the title row (secondary). Empty when unset.
    public static func formatTemplateDescription(_ template: TemplatesPickerItem) -> String {
        template.description.trimmingCharacters(in: .whitespacesAndNewlines)
    }

    /// Second-line body preview: whitespace collapsed, truncated. Path if body empty.
    public static func formatTemplateBodyPreview(_ template: TemplatesPickerItem) -> String {
        let collapsed = collapseWhitespace(template.body)
        if collapsed.isEmpty { return template.path }
        if collapsed.count <= templateBodyPreviewMaxChars { return collapsed }
        let keep = max(0, templateBodyPreviewMaxChars - 1)
        return String(collapsed.prefix(keep)) + "…"
    }

    /// Second list line for templates (body preview).
    public static func formatTemplateSubtitle(_ template: TemplatesPickerItem) -> String {
        formatTemplateBodyPreview(template)
    }

    /// Collapse runs of whitespace/newlines to a single space for one-line preview.
    public static func collapseWhitespace(_ text: String) -> String {
        let parts = text.split { $0.isWhitespace || $0.isNewline }.filter { !$0.isEmpty }
        return parts.joined(separator: " ")
    }

    /// Body fuzzy spans for the preview line when they still align (no newline collapse).
    public static func formatTemplateBodyPreviewSpans(_ template: TemplatesPickerItem) -> [FuzzySpan] {
        if template.bodySpans.isEmpty { return [] }
        let trimmed = template.body.trimmingCharacters(in: .whitespacesAndNewlines)
        if collapseWhitespace(template.body) != trimmed { return [] }
        return template.bodySpans
    }

    public static func formatFileTitle(_ file: FilesPickerItem) -> String {
        let note = file.note.trimmingCharacters(in: .whitespacesAndNewlines)
        if !note.isEmpty { return note }
        let name = file.name.trimmingCharacters(in: .whitespacesAndNewlines)
        if !name.isEmpty { return name }
        return URL(fileURLWithPath: file.path).lastPathComponent
    }

    public static func formatFileSubtitle(_ file: FilesPickerItem) -> String {
        if file.exists {
            return file.path
        }
        return "\(file.path)  (missing)"
    }

    public static func formatUseCount(_ n: Int) -> String {
        n > 0 ? "\(n)" : ""
    }

    public static func formatEmptyTitle(sidebarID: String) -> String {
        switch normalizeSidebarID(sidebarID) {
        case sidebarTemplates:
            return "No templates registered"
        case sidebarSkills:
            return "No skills registered"
        case sidebarFiles:
            return "No files registered"
        default:
            return "Nothing registered"
        }
    }

    public static func formatEmptyHint(sidebarID: String) -> String {
        switch normalizeSidebarID(sidebarID) {
        case sidebarTemplates:
            return "register a root with: my templates --add-dir"
        case sidebarSkills:
            return "register a root with: my skills --add-dir"
        case sidebarFiles:
            return "register with: my files --add"
        default:
            return "register with: my skills / my templates / my files"
        }
    }

    public static func formatNoResults() -> String {
        "No Results"
    }

    public static func formatSearchPrompt(sidebarID: String) -> String {
        switch normalizeSidebarID(sidebarID) {
        case sidebarTemplates:
            return "Search templates"
        case sidebarSkills:
            return "Search skills"
        case sidebarFiles:
            return "Search files"
        default:
            return "Search skills, templates & files"
        }
    }

    /// Debounce for list GETs (lodash-style trailing).
    public static let searchDebounceNanoseconds: UInt64 = 150_000_000

    /// Server spans when present; otherwise one unmatched fallback (title/path).
    public static func displaySpans(_ spans: [FuzzySpan], fallback: String) -> [FuzzySpan] {
        if spans.isEmpty {
            return [FuzzySpan(text: fallback, matched: false)]
        }
        return spans
    }

    public static func joinSpans(_ spans: [FuzzySpan]) -> String {
        spans.map(\.text).joined()
    }

    /// Debounce aborts in-flight GET; that is not a user-facing failure.
    public static func isIgnorableSearchError(_ error: Error) -> Bool {
        if error is CancellationError {
            return true
        }
        if let url = error as? URLError, url.code == .cancelled {
            return true
        }
        let ns = error as NSError
        return ns.domain == NSURLErrorDomain && ns.code == NSURLErrorCancelled
    }

    /// Merge ranked skill + template + file lists for the All sidebar (score/useCount desc).
    public static func mergeAllItems(
        skills: [SkillsPickerItem],
        templates: [TemplatesPickerItem],
        files: [FilesPickerItem] = []
    ) -> [InsertPickerItem] {
        var out: [InsertPickerItem] = []
        out.reserveCapacity(skills.count + templates.count + files.count)
        for s in skills {
            out.append(InsertPickerItem(kind: .skill, skill: s, template: nil, file: nil))
        }
        for t in templates {
            out.append(InsertPickerItem(kind: .template, skill: nil, template: t, file: nil))
        }
        for f in files {
            out.append(InsertPickerItem(kind: .file, skill: nil, template: nil, file: f))
        }
        out.sort { a, b in
            let sa = a.score
            let sb = b.score
            if sa != sb { return sa > sb }
            let ua = a.useCount
            let ub = b.useCount
            if ua != ub { return ua > ub }
            return a.title.localizedCaseInsensitiveCompare(b.title) == .orderedAscending
        }
        return out
    }
}

public enum InsertPickerKind: String, Equatable {
    case skill
    case template
    case file
}

public struct InsertPickerItem: Equatable, Identifiable {
    public var id: String { "\(kind.rawValue):\(path)" }
    public let kind: InsertPickerKind
    public let skill: SkillsPickerItem?
    public let template: TemplatesPickerItem?
    public let file: FilesPickerItem?

    public var path: String {
        switch kind {
        case .skill: return skill?.path ?? ""
        case .template: return template?.path ?? ""
        case .file: return file?.path ?? ""
        }
    }

    public var title: String {
        switch kind {
        case .skill:
            return skill.map(SkillsPickerFormatter.formatTitle) ?? ""
        case .template:
            return template.map(SkillsPickerFormatter.formatTemplateTitle) ?? ""
        case .file:
            return file.map(SkillsPickerFormatter.formatFileTitle) ?? ""
        }
    }

    public var subtitle: String {
        switch kind {
        case .skill:
            return skill.map(SkillsPickerFormatter.formatSubtitle) ?? ""
        case .template:
            return template.map(SkillsPickerFormatter.formatTemplateSubtitle) ?? ""
        case .file:
            return file.map(SkillsPickerFormatter.formatFileSubtitle) ?? ""
        }
    }

    /// Trailing description on the title row (templates only).
    public var trailingDescription: String {
        switch kind {
        case .template:
            return template.map(SkillsPickerFormatter.formatTemplateDescription) ?? ""
        case .skill, .file:
            return ""
        }
    }

    public var useCount: Int {
        switch kind {
        case .skill: return skill?.useCount ?? 0
        case .template: return template?.useCount ?? 0
        case .file: return file?.useCount ?? 0
        }
    }

    public var score: Int {
        switch kind {
        case .skill: return 0
        case .template: return template?.score ?? 0
        case .file: return file?.score ?? 0
        }
    }

    public var titleSpans: [FuzzySpan] {
        switch kind {
        case .skill: return skill?.titleSpans ?? []
        case .template: return template?.titleSpans ?? []
        case .file: return file?.titleSpans ?? []
        }
    }

    public var pathSpans: [FuzzySpan] {
        switch kind {
        case .skill: return skill?.pathSpans ?? []
        case .template: return template?.pathSpans ?? []
        case .file: return file?.pathSpans ?? []
        }
    }

    /// Body highlight spans for the template preview line.
    public var bodySpans: [FuzzySpan] {
        switch kind {
        case .template:
            return template.map(SkillsPickerFormatter.formatTemplateBodyPreviewSpans) ?? []
        case .skill, .file:
            return []
        }
    }

    public var clipboardText: String {
        switch kind {
        case .skill:
            return skill?.path ?? ""
        case .template:
            return template?.body ?? ""
        case .file:
            return file?.path ?? ""
        }
    }

    public init(kind: InsertPickerKind, skill: SkillsPickerItem?, template: TemplatesPickerItem?, file: FilesPickerItem? = nil) {
        self.kind = kind
        self.skill = skill
        self.template = template
        self.file = file
    }
}
