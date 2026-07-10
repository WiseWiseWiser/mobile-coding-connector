import Foundation

/// Projects submenu labels — mirrors `macosapp/menubar` project formatters.
public enum ProjectsMenuFormatter {
    public static func formatProjectTitle(name: String, branch: String, clean: Bool, errMsg: String) -> String {
        if !errMsg.isEmpty {
            return "\(name) ⚠ Error"
        }
        if clean {
            return "\(name) ● \(branch)"
        }
        return "\(name) ○ \(branch)"
    }

    public static func formatWorktreeTitle(name: String, clean: Bool) -> String {
        if clean {
            return "\(name) ● Clean"
        }
        return "\(name) ○ Dirty"
    }

    public static func formatProjectsEmptyLabel() -> String {
        "No wrk projects"
    }
}
