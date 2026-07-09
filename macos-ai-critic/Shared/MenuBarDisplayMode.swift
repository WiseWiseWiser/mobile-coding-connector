import Foundation

/// Shared menu-bar title display mode (Grok / Codex / rotating).
public enum MenuBarDisplayMode: String, CaseIterable, Identifiable {
    case rotating
    case grok
    case codex

    public var id: String { rawValue }

    public var displayName: String {
        switch self {
        case .rotating:
            return "Rotating"
        case .grok:
            return "Grok"
        case .codex:
            return "Codex"
        }
    }
}
