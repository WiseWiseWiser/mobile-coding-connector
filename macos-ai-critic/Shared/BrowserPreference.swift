import Foundation

public enum BrowserPreference: String, CaseIterable, Identifiable {
    case `default` = "default"
    case chrome = "chrome"
    case firefox = "firefox"
    case opera = "opera"

    public var id: String { rawValue }

    public var displayName: String {
        switch self {
        case .default: return "Default"
        case .chrome: return "Chrome"
        case .firefox: return "Firefox"
        case .opera: return "Opera"
        }
    }

    public static func fromStored(_ value: String) -> BrowserPreference {
        BrowserPreference(rawValue: value) ?? .default
    }
}
