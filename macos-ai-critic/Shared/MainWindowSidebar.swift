import Foundation

/// Sidebar destinations for the main AI Critic window (mirrors macosapp/mainwindow).
public enum MainSidebarItem: String, CaseIterable, Identifiable, Hashable {
    case home
    case services
    case projects
    case settings

    public var id: String { rawValue }

    public var title: String {
        MainWindowFormatter.formatSidebarTitle(id: rawValue)
    }

    public var systemImage: String {
        switch self {
        case .home: return "house"
        case .services: return "server.rack"
        case .projects: return "folder"
        case .settings: return "gearshape"
        }
    }
}

/// Pure labels for the Show App window (mirrors macosapp/mainwindow).
public enum MainWindowFormatter {
    public static let storageKey = "mainSidebarPage"

    public static func formatSidebarTitle(id: String) -> String {
        switch id {
        case "home": return "Home"
        case "services": return "Services"
        case "projects": return "Projects"
        case "settings": return "Settings"
        default: return ""
        }
    }

    public static func formatShowAppLabel() -> String {
        "Show App"
    }

    public static func normalizeSidebarID(_ id: String) -> String {
        switch id {
        case "home", "services", "projects", "settings":
            return id
        default:
            return "home"
        }
    }
}
