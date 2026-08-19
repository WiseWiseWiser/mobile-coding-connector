import Foundation

/// Pure formatters and fuzzy ranking for the skills picker.
public enum SkillsPickerFormatter {
    public static func formatWindowTitle() -> String {
        "Skills"
    }

    public static func formatHotKey() -> String {
        "⌘⇧;"
    }

    public static func formatCopiedToast() -> String {
        "Copied"
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

    public static func formatUseCount(_ n: Int) -> String {
        n > 0 ? "\(n)" : ""
    }

    public static func formatEmptyTitle() -> String {
        "No skills registered"
    }

    public static func formatEmptyHint() -> String {
        "register a root with: my skills --add-dir"
    }

    public static func formatNoResults() -> String {
        "No Results"
    }

    public static func formatSearchPrompt() -> String {
        "Search skills"
    }

    /// Debounce for GET /api/local/skills?q= (lodash-style trailing).
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
}
