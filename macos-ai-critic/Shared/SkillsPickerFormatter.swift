import Foundation

/// Pure formatters for the ⌘⇧; insert picker (skills + templates + files + commands).
public enum SkillsPickerFormatter {
    public static let sidebarAll = "all"
    public static let sidebarSkills = "skills"
    public static let sidebarTemplates = "templates"
    public static let sidebarFiles = "files"
    public static let sidebarCommands = "commands"
    public static let sidebarClipboard = "clipboard"
    public static let sidebarAdhoc = "adhoc"
    public static let sidebarConvert = "convert"
    public static let sidebarDefaultsKey = "insertPickerSidebarID"
    public static let sidebarWidthDefaultsKey = "insertPickerSidebarWidth"
    /// Persisted clipboard "Copy file path" prefix (as typed, including trailing space).
    public static let clipboardPathPrefixDefaultsKey = "insertPickerClipboardPathPrefix"
    /// Persisted: when true, Copy file path appends OCR text after the dumped path.
    public static let clipboardAppendOCRDefaultsKey = "insertPickerClipboardAppendOCR"

    /// Ordered sidebar ids (All → Commands, then non-search Clipboard / Adhoc / Convert).
    public static let sidebarOrder: [String] = [
        sidebarAll, sidebarSkills, sidebarTemplates, sidebarFiles, sidebarCommands,
        sidebarClipboard, sidebarAdhoc, sidebarConvert,
    ]

    /// Debounce for adhoc text PUT (trailing).
    public static let adhocSaveDebounceNanoseconds: UInt64 = 400_000_000
    /// Debounce for convert-text preview (trailing).
    public static let convertDebounceNanoseconds: UInt64 = 400_000_000

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
        case sidebarCommands: return "terminal"
        case sidebarClipboard: return "doc.on.clipboard"
        case sidebarAdhoc: return "note.text"
        case sidebarConvert: return "arrow.triangle.2.circlepath"
        default: return "square.grid.2x2"
        }
    }

    /// Searchable sidebars hit list APIs; clipboard/adhoc/convert do not.
    public static func isSearchableSidebar(_ id: String) -> Bool {
        switch normalizeSidebarID(id) {
        case sidebarClipboard, sidebarAdhoc, sidebarConvert:
            return false
        default:
            return true
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
        case sidebarCommands: return "Commands"
        case sidebarClipboard: return "Clipboard"
        case sidebarAdhoc: return "Adhoc text"
        case sidebarConvert: return "Convert text"
        default: return ""
        }
    }

    public static func normalizeSidebarID(_ id: String?) -> String {
        let trimmed = (id ?? "").trimmingCharacters(in: .whitespacesAndNewlines)
        switch trimmed {
        case sidebarAll, sidebarSkills, sidebarTemplates, sidebarFiles, sidebarCommands,
             sidebarClipboard, sidebarAdhoc, sidebarConvert:
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
        case .command: return "CMD"
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

    public static func formatCommandTitle(_ command: CommandsPickerItem) -> String {
        let note = command.note.trimmingCharacters(in: .whitespacesAndNewlines)
        if !note.isEmpty { return note }
        let name = command.name.trimmingCharacters(in: .whitespacesAndNewlines)
        if !name.isEmpty { return name }
        return command.command
    }

    public static func formatCommandSubtitle(_ command: CommandsPickerItem) -> String {
        command.command
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
        case sidebarCommands:
            return "No commands registered"
        case sidebarClipboard:
            return "Clipboard empty"
        case sidebarAdhoc:
            return "Adhoc text"
        case sidebarConvert:
            return "Convert text"
        default:
            return "Nothing registered"
        }
    }

    public static func formatEmptyHint(sidebarID: String) -> String {
        switch normalizeSidebarID(sidebarID) {
        case sidebarTemplates:
            return "Add a template below, or: my templates --add-dir"
        case sidebarSkills:
            return "register a root with: my skills --add-dir"
        case sidebarFiles:
            return "Add a file below, or: my files --add"
        case sidebarCommands:
            return "Add a command below, or: my commands --add"
        case sidebarClipboard:
            return "Copy something, then Refresh"
        case sidebarAdhoc:
            return "Compose temporary text; auto-saves"
        case sidebarConvert:
            return "Paste multiline text; converted preview appears after you pause typing"
        default:
            return "register with: my skills / my templates / my files / my commands"
        }
    }

    public static func formatClipboardHeading() -> String { "Clipboard" }
    public static func formatAdhocHeading() -> String { "Adhoc text" }
    public static func formatConvertHeading() -> String { "Convert text" }
    public static func formatDumpToFileTitle() -> String { "Dump to file" }
    public static func formatCopyFilePathTitle() -> String { "Copy file path" }
    public static func formatCopyTextTitle() -> String { "Copy text" }
    public static func formatRefreshTitle() -> String { "Refresh" }
    public static func formatLastDumpLabel() -> String { "Last dump:" }
    public static func formatClipboardKindLabel() -> String { "Kind:" }
    public static func formatClipboardSizeLabel() -> String { "Size:" }
    public static func formatPathPrefixLabel() -> String { "Path prefix" }
    public static func formatPathPrefixPlaceholder() -> String { "optional, e.g. image " }
    public static func formatCopyWillUseLabel() -> String { "Copy will use:" }
    public static func formatOCRHeading() -> String { "OCR" }
    public static func formatOCRCheckboxTitle() -> String { "OCR" }
    public static func formatOCRCheckboxHelp() -> String { "Append OCR text after path when copying" }
    public static func formatOCRRecognizing() -> String { "Recognizing…" }
    public static func formatOCREmpty() -> String { "(no text recognized)" }
    public static func formatAdhocSavedStatus() -> String { "Saved" }
    public static func formatAdhocSavingStatus() -> String { "Saving…" }
    public static func formatAdhocDirtyStatus() -> String { "Unsaved" }
    public static func formatConvertResultHeading() -> String { "Converted" }
    public static func formatConvertEmptyHint() -> String { "Converted preview appears after you pause typing" }
    public static func formatConvertingStatus() -> String { "Converting…" }
    public static func formatCopyConvertedTextTitle() -> String { "Copy converted text" }
    public static func formatPathCopiedToast() -> String { "Path copied" }

    /// Pasteboard string for Copy file path: prefix as typed + path (no auto space).
    /// Empty path → empty string. Empty prefix → path only.
    /// When `appendOCR` and `ocrText` is non-empty, appends `\n` + trimmed OCR after the path line.
    public static func formatClipboardCopyText(
        prefix: String,
        path: String,
        appendOCR: Bool = false,
        ocrText: String = ""
    ) -> String {
        let trimmedPath = path.trimmingCharacters(in: .whitespacesAndNewlines)
        if trimmedPath.isEmpty { return "" }
        let base: String
        if prefix.isEmpty {
            base = trimmedPath
        } else {
            base = prefix + trimmedPath
        }
        guard appendOCR else { return base }
        let ocr = ocrText.trimmingCharacters(in: .whitespacesAndNewlines)
        if ocr.isEmpty { return base }
        return base + "\n" + ocr
    }

    /// Live preview line under the prefix field (empty path → empty).
    /// Unused in the Clipboard UI (too much chrome); kept for tests / callers.
    public static func formatCopyWillUsePreview(prefix: String, path: String) -> String {
        let text = formatClipboardCopyText(prefix: prefix, path: path)
        if text.isEmpty { return "" }
        return "\(formatCopyWillUseLabel()) \(text)"
    }

    public static func isClipboardImageKind(_ kind: String) -> Bool {
        kind.trimmingCharacters(in: .whitespacesAndNewlines).lowercased() == "image"
    }

    public static func formatClipboardKind(_ kind: String) -> String {
        let k = kind.trimmingCharacters(in: .whitespacesAndNewlines).lowercased()
        switch k {
        case "", "empty": return "empty"
        case "text": return "text"
        case "image": return "image"
        case "html": return "html"
        case "svg": return "svg"
        case "rtf": return "rtf"
        case "pdf": return "pdf"
        case "unsupported": return "unsupported"
        default: return k
        }
    }

    public static func formatClipboardSize(bytes: Int) -> String {
        if bytes <= 0 { return "0 B" }
        let units = ["B", "KB", "MB", "GB"]
        var value = Double(bytes)
        var i = 0
        while value >= 1024 && i < units.count - 1 {
            value /= 1024
            i += 1
        }
        if i == 0 {
            return "\(bytes) B"
        }
        return String(format: "%.1f %@", value, units[i])
    }

    public static func canDumpClipboard(kind: String) -> Bool {
        switch kind.trimmingCharacters(in: .whitespacesAndNewlines).lowercased() {
        case "", "empty", "unsupported":
            return false
        default:
            return true
        }
    }

    public static func formatAddButtonTitle(sidebarID: String) -> String {
        switch normalizeSidebarID(sidebarID) {
        case sidebarTemplates:
            return "New template"
        case sidebarFiles:
            return "Add file or folder"
        case sidebarCommands:
            return "Add command"
        default:
            return ""
        }
    }

    public static func shouldShowAddButton(sidebarID: String) -> Bool {
        switch normalizeSidebarID(sidebarID) {
        case sidebarTemplates, sidebarFiles, sidebarCommands:
            return true
        default:
            return false
        }
    }

    public static func formatChooseTemplateFolderTitle() -> String {
        "Choose template folder…"
    }

    public static func formatNewTemplateSheetTitle() -> String {
        "New template"
    }

    public static func formatAddFileSheetTitle() -> String {
        "Add file or folder"
    }

    public static func formatAddCommandSheetTitle() -> String {
        "Add command"
    }

    /// Flat .md basename from a display name (mirrors server slugify).
    public static func slugifyTemplateFilename(_ name: String) -> String {
        var s = name.trimmingCharacters(in: .whitespacesAndNewlines)
        s = String(s.map { ch -> Character in
            ch.isWhitespace ? "-" : ch
        })
        let allowed = CharacterSet.alphanumerics.union(CharacterSet(charactersIn: "._-"))
        s = String(s.unicodeScalars.map { scalar -> Character in
            allowed.contains(scalar) ? Character(scalar) : "-"
        })
        while s.contains("--") {
            s = s.replacingOccurrences(of: "--", with: "-")
        }
        s = s.trimmingCharacters(in: CharacterSet(charactersIn: "-._"))
        if s.isEmpty { s = "template" }
        if !s.lowercased().hasSuffix(".md") {
            s += ".md"
        }
        return s
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
        case sidebarCommands:
            return "Search commands"
        case sidebarClipboard, sidebarAdhoc, sidebarConvert:
            return ""
        default:
            return "Search skills, templates, files & commands"
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

    /// Merge ranked skill + template + file + command lists for the All sidebar (score/useCount desc).
    public static func mergeAllItems(
        skills: [SkillsPickerItem],
        templates: [TemplatesPickerItem],
        files: [FilesPickerItem] = [],
        commands: [CommandsPickerItem] = []
    ) -> [InsertPickerItem] {
        var out: [InsertPickerItem] = []
        out.reserveCapacity(skills.count + templates.count + files.count + commands.count)
        for s in skills {
            out.append(InsertPickerItem(kind: .skill, skill: s, template: nil, file: nil, command: nil))
        }
        for t in templates {
            out.append(InsertPickerItem(kind: .template, skill: nil, template: t, file: nil, command: nil))
        }
        for f in files {
            out.append(InsertPickerItem(kind: .file, skill: nil, template: nil, file: f, command: nil))
        }
        for c in commands {
            out.append(InsertPickerItem(kind: .command, skill: nil, template: nil, file: nil, command: c))
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
    case command
}

public struct InsertPickerItem: Equatable, Identifiable {
    public var id: String { "\(kind.rawValue):\(path)" }
    public let kind: InsertPickerKind
    public let skill: SkillsPickerItem?
    public let template: TemplatesPickerItem?
    public let file: FilesPickerItem?
    public let command: CommandsPickerItem?

    public var path: String {
        switch kind {
        case .skill: return skill?.path ?? ""
        case .template: return template?.path ?? ""
        case .file: return file?.path ?? ""
        case .command: return command?.command ?? ""
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
        case .command:
            return command.map(SkillsPickerFormatter.formatCommandTitle) ?? ""
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
        case .command:
            return command.map(SkillsPickerFormatter.formatCommandSubtitle) ?? ""
        }
    }

    /// Trailing description on the title row (templates only).
    public var trailingDescription: String {
        switch kind {
        case .template:
            return template.map(SkillsPickerFormatter.formatTemplateDescription) ?? ""
        case .skill, .file, .command:
            return ""
        }
    }

    public var useCount: Int {
        switch kind {
        case .skill: return skill?.useCount ?? 0
        case .template: return template?.useCount ?? 0
        case .file: return file?.useCount ?? 0
        case .command: return command?.useCount ?? 0
        }
    }

    public var score: Int {
        switch kind {
        case .skill: return 0
        case .template: return template?.score ?? 0
        case .file: return file?.score ?? 0
        case .command: return command?.score ?? 0
        }
    }

    public var titleSpans: [FuzzySpan] {
        switch kind {
        case .skill: return skill?.titleSpans ?? []
        case .template: return template?.titleSpans ?? []
        case .file: return file?.titleSpans ?? []
        case .command: return command?.titleSpans ?? []
        }
    }

    public var pathSpans: [FuzzySpan] {
        switch kind {
        case .skill: return skill?.pathSpans ?? []
        case .template: return template?.pathSpans ?? []
        case .file: return file?.pathSpans ?? []
        case .command: return command?.commandSpans ?? []
        }
    }

    /// Body highlight spans for the template preview line.
    public var bodySpans: [FuzzySpan] {
        switch kind {
        case .template:
            return template.map(SkillsPickerFormatter.formatTemplateBodyPreviewSpans) ?? []
        case .skill, .file, .command:
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
        case .command:
            return command?.command ?? ""
        }
    }

    public init(
        kind: InsertPickerKind,
        skill: SkillsPickerItem?,
        template: TemplatesPickerItem?,
        file: FilesPickerItem? = nil,
        command: CommandsPickerItem? = nil
    ) {
        self.kind = kind
        self.skill = skill
        self.template = template
        self.file = file
        self.command = command
    }
}
