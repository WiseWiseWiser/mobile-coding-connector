import Foundation

/// Pure formatters for the Bookmarks menu (mirrors server/bookmarks labels).
public enum BookmarkMenuFormatter {
    public static func formatEmptyBookmarksLabel() -> String {
        "No bookmarks"
    }

    public static func formatBookmarkMenuTitle(name: String) -> String {
        name
    }

    /// Resolve effective browser: non-empty bookmark browser wins, else global default, else "default".
    public static func resolveBrowser(bookmarkBrowser: String?, globalDefault: String) -> String {
        if let b = bookmarkBrowser?.trimmingCharacters(in: .whitespacesAndNewlines), !b.isEmpty {
            return b
        }
        let g = globalDefault.trimmingCharacters(in: .whitespacesAndNewlines)
        if !g.isEmpty {
            return g
        }
        return "default"
    }
}
