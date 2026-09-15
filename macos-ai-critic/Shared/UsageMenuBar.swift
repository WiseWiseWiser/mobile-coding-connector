import Foundation

/// Chooses which registered usage item the menu bar shows. All display text
/// comes from the server; this type only picks the item and rotates.
public enum UsageMenuBar {
    /// Title shown while the first fetch is still in flight.
    public static let fallbackTitle = "Usage ..."

    /// Enabled items in registry order.
    public static func enabledItems(_ response: UsageItemsResponse?) -> [UsageItemView] {
        (response?.items ?? []).filter { $0.enabled }
    }

    /// The item whose title the menu bar shows: the pinned default, or — while the
    /// server has rotation on — the item at `rotatingIndex` steps after it.
    public static func selectedItem(
        _ response: UsageItemsResponse?,
        rotatingIndex: Int
    ) -> UsageItemView? {
        guard let response else { return nil }
        let enabled = enabledItems(response)
        guard !enabled.isEmpty else { return nil }
        if response.rotate {
            // Rotation starts at the default item so `--rotate --start <id>` shows it first.
            let start = enabled.firstIndex { $0.id == response.defaultID } ?? 0
            let steps = rotatingIndex < 0 ? 0 : rotatingIndex
            return enabled[(start + steps) % enabled.count]
        }
        return enabled.first { $0.id == response.defaultID } ?? enabled.first
    }

    /// Menu-bar title, printed verbatim from the server.
    public static func title(_ response: UsageItemsResponse?, rotatingIndex: Int) -> String {
        guard let item = selectedItem(response, rotatingIndex: rotatingIndex) else {
            return fallbackTitle
        }
        let trimmed = item.title.trimmingCharacters(in: .whitespacesAndNewlines)
        return trimmed.isEmpty ? fallbackTitle : trimmed
    }

    /// Dropdown lines for every enabled item, printed verbatim from the server.
    public static func dropdownLines(_ response: UsageItemsResponse?) -> [String] {
        enabledItems(response)
            .map(\.dropdown)
            .filter { !$0.trimmingCharacters(in: .whitespacesAndNewlines).isEmpty }
    }

    /// Next rotation cursor over the enabled items.
    public static func nextRotatingIndex(_ response: UsageItemsResponse?, current: Int) -> Int {
        let count = enabledItems(response).count
        guard count > 0 else { return 0 }
        return (max(current, 0) + 1) % count
    }
}
