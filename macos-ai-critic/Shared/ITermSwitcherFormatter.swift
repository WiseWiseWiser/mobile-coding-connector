import Foundation

/// Pure formatters for the iTerm switcher (mirrors macosapp/itermswitcher).
public enum ITermSwitcherFormatter {
    public static func formatSessionPrimary(name: String, cwd: String, sessionID: String) -> String {
        let trimmed = name.trimmingCharacters(in: .whitespacesAndNewlines)
        if !trimmed.isEmpty { return trimmed }
        let cwdTrimmed = cwd.trimmingCharacters(in: .whitespacesAndNewlines)
        if !cwdTrimmed.isEmpty { return cwdTrimmed }
        return shortSessionID(sessionID)
    }

    public static func formatSessionNote(_ note: String) -> String {
        let trimmed = note.trimmingCharacters(in: .whitespacesAndNewlines)
        return trimmed.isEmpty ? "—" : trimmed
    }

    public static func formatSessionTitle(name: String, note: String, cwd: String, sessionID: String) -> String {
        let primary = formatSessionPrimary(name: name, cwd: cwd, sessionID: sessionID)
        let n = formatSessionNote(note)
        if n == "—" { return primary }
        return primary + "  ·  " + n
    }

    public static func formatDesktopHeader(spaceIndex: Int) -> String {
        let idx = spaceIndex < 0 ? 0 : spaceIndex
        return "Desktop \(idx + 1)"
    }

    public static func formatSidebarDesktopTitle(spaceIndex: Int, label: String) -> String {
        let trimmed = label.trimmingCharacters(in: .whitespacesAndNewlines)
        if !trimmed.isEmpty { return trimmed }
        return formatDesktopHeader(spaceIndex: spaceIndex)
    }

    public static func formatDesktopSidebarSymbol(current: Bool) -> String {
        // macwindow.fill is not a real SF Symbol (blank icon). Same glyph; current is tinted in the view.
        _ = current
        return "macwindow"
    }

    public static func formatSpaceLabelRow(_ label: String) -> String {
        let trimmed = label.trimmingCharacters(in: .whitespacesAndNewlines)
        return trimmed.isEmpty ? "Set Space Label" : trimmed
    }

    public static func formatChangeSpaceLabel() -> String {
        "Change"
    }

    public static func formatClearSpaceLabel() -> String {
        "Clear"
    }

    public static func formatShowSpaceLabel() -> String {
        "Show"
    }

    public static func formatEditSpaceLabel() -> String {
        "Edit"
    }

    public static func formatDismissSpaceLabel() -> String {
        "Dismiss"
    }

    public static let spaceLabelRowID = "__space-label__"

    public static func formatEmptyITerm() -> String {
        "iTerm2 is not running"
    }

    public static func formatSavedNotesHeader(count: Int) -> String {
        let n = count < 0 ? 0 : count
        return "Saved notes (\(n))"
    }

    public static func formatBusyLabel(idle: Bool?) -> String {
        guard let idle else { return "" }
        return idle ? "idle" : "busy"
    }

    public static func formatSessionSubtitle(cwd: String, tabIndex: Int, idle: Bool?) -> String {
        var parts: [String] = []
        let cwdTrimmed = cwd.trimmingCharacters(in: .whitespacesAndNewlines)
        if !cwdTrimmed.isEmpty { parts.append(cwdTrimmed) }
        if tabIndex > 0 { parts.append("tab \(tabIndex)") }
        let busy = formatBusyLabel(idle: idle)
        if !busy.isEmpty { parts.append(busy) }
        return parts.joined(separator: "  ·  ")
    }

    public static func shortSessionID(_ id: String) -> String {
        let trimmed = id.trimmingCharacters(in: .whitespacesAndNewlines)
        if trimmed.isEmpty { return "" }
        if let dash = trimmed.firstIndex(of: "-") {
            return String(trimmed[..<dash])
        }
        if trimmed.count > 8 {
            return String(trimmed.prefix(8))
        }
        return trimmed
    }

    public static func formatDefaultHotKey() -> String {
        "⌘⇧Space"
    }

    public static let sidebarAll = "all"
    public static let sidebarBookmarks = "bookmarks"
    public static let sidebarSaved = "saved"

    public static func formatWindowTitle() -> String {
        "Terminals"
    }

    public static func formatBookmarksHeader() -> String {
        formatSidebarTitle(id: sidebarBookmarks)
    }

    /// Sidebar id to select when the switcher opens: current Desktop, else All.
    public static func initialSidebarID(currentSpaceIndex: Int?) -> String {
        guard let idx = currentSpaceIndex, idx >= 0 else { return sidebarAll }
        return sidebarDesktopID(spaceIndex: idx)
    }

    public static func sidebarDesktopID(spaceIndex: Int) -> String {
        let idx = spaceIndex < 0 ? 0 : spaceIndex
        return "desktop:\(idx)"
    }

    public static func parseSidebarDesktop(_ id: String) -> Int? {
        let trimmed = id.trimmingCharacters(in: .whitespacesAndNewlines)
        guard trimmed.hasPrefix("desktop:") else { return nil }
        let rest = trimmed.dropFirst("desktop:".count)
        guard let n = Int(rest), n >= 0 else { return nil }
        return n
    }

    /// True only for Per-Space sidebar rows. All / Bookmarks / Saved do not switch.
    public static func shouldSwitchSpace(sidebarID: String) -> Bool {
        parseSidebarDesktop(sidebarID) != nil
    }

    public static func formatSwitchSpaceMissingID() -> String {
        "Can't switch Desktop — space id is missing"
    }

    public static func formatSwitchSpaceFailed() -> String {
        "Can't switch Desktop"
    }

    public static func formatSidebarTitle(id: String) -> String {
        switch id.trimmingCharacters(in: .whitespacesAndNewlines) {
        case "", sidebarAll: return "All"
        case sidebarBookmarks: return "Bookmarks"
        case sidebarSaved: return "Saved notes"
        default:
            if let idx = parseSidebarDesktop(id) {
                return formatDesktopHeader(spaceIndex: idx)
            }
            return ""
        }
    }

    public static func matchesSidebar(id: String, spaceIndex: Int, bookmarked: Bool) -> Bool {
        switch id.trimmingCharacters(in: .whitespacesAndNewlines) {
        case "", sidebarAll: return true
        case sidebarBookmarks: return bookmarked
        case sidebarSaved: return false
        default:
            if let idx = parseSidebarDesktop(id) {
                let space = spaceIndex < 0 ? 0 : spaceIndex
                return idx == space
            }
            return true
        }
    }

    public static func formatOrphanPrimary(note: String, sessionName: String, cwd: String, sessionID: String) -> String {
        let trimmed = note.trimmingCharacters(in: .whitespacesAndNewlines)
        if !trimmed.isEmpty { return trimmed }
        return formatSessionPrimary(name: sessionName, cwd: cwd, sessionID: sessionID)
    }

    public static func resolvedBookmarked(sessionID: String, inventoryValue: Bool, overrides: [String: Bool]) -> Bool {
        overrides[sessionID] ?? inventoryValue
    }

    public static func reconcileBookmarkOverrides(overrides: [String: Bool], live: [String: Bool]) -> [String: Bool] {
        var out: [String: Bool] = [:]
        for (id, want) in overrides {
            guard let cur = live[id], cur != want else { continue }
            out[id] = want
        }
        return out
    }

    public static func formatHotKey(keyCode: Int, modifiers: Int) -> String {
        var parts = ""
        if modifiers & 4096 != 0 { parts += "⌃" }
        if modifiers & 2048 != 0 { parts += "⌥" }
        if modifiers & 256 != 0 { parts += "⌘" }
        if modifiers & 512 != 0 { parts += "⇧" }
        parts += keyName(keyCode: keyCode)
        return parts
    }

    public static func sessionMatches(
        name: String,
        note: String,
        cwd: String,
        windowName: String,
        tabName: String,
        sessionID: String,
        spaceIndex: Int,
        spaceLabel: String = "",
        query: String
    ) -> Bool {
        let q = query.trimmingCharacters(in: .whitespacesAndNewlines).lowercased()
        if q.isEmpty { return true }
        let desktop = formatSidebarDesktopTitle(spaceIndex: spaceIndex, label: spaceLabel).lowercased()
        let hay = [name, note, cwd, windowName, tabName, desktop, spaceLabel, sessionID]
            .joined(separator: " ")
            .lowercased()
        return hay.contains(q)
    }

    public static func orphanMatches(_ orphan: ITermOrphanNote, query: String) -> Bool {
        sessionMatches(
            name: orphan.sessionName,
            note: orphan.note,
            cwd: orphan.cwd,
            windowName: orphan.windowName,
            tabName: "",
            sessionID: orphan.sessionID,
            spaceIndex: orphan.spaceIndex,
            query: query
        )
    }

    private static func keyName(keyCode: Int) -> String {
        switch keyCode {
        case 49: return "Space"
        case 36: return "Return"
        case 48: return "Tab"
        case 53: return "Esc"
        case 51: return "Delete"
        default:
            return "Key\(keyCode)"
        }
    }
}
