import Foundation

/// Shared sidebar selection so menu "Show App" / "Settings…" can pick a page.
@MainActor
public final class MainWindowRouter: ObservableObject {
    public static let shared = MainWindowRouter()

    @Published public var selection: MainSidebarItem {
        didSet {
            UserDefaults.standard.set(selection.rawValue, forKey: MainWindowFormatter.storageKey)
        }
    }

    private init() {
        let raw = UserDefaults.standard.string(forKey: MainWindowFormatter.storageKey) ?? ""
        let id = MainWindowFormatter.normalizeSidebarID(raw)
        selection = MainSidebarItem(rawValue: id) ?? .home
    }

    public func open(_ page: MainSidebarItem) {
        selection = page
    }
}
