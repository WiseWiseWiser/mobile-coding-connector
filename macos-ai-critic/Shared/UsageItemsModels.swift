import Foundation

/// One usage source as the server renders it. `title` is the menu-bar label and
/// `dropdown` the dropdown line; the app shows both verbatim.
public struct UsageItemView: Decodable, Identifiable, Equatable {
    public let id: String
    public let label: String
    public let kind: String
    public let enabled: Bool
    public let status: String
    public let title: String
    public let dropdown: String
    public let detail: String?
    public let usageURL: String?
    public let error: String?
    public let updatedAt: String?

    enum CodingKeys: String, CodingKey {
        case id, label, kind, enabled, status, title, dropdown, detail, error
        case usageURL = "usage_url"
        case updatedAt = "updated_at"
    }
}

/// `GET /api/usage/items`: every item plus the menu-bar selection state.
public struct UsageItemsResponse: Decodable, Equatable {
    public let version: Int
    /// Id of the pinned menu-bar item when rotation is off. `default` is a Swift
    /// keyword, so the JSON key is mapped explicitly.
    public let defaultID: String
    public let rotate: Bool
    public let items: [UsageItemView]

    enum CodingKeys: String, CodingKey {
        case version, rotate, items
        case defaultID = "default"
    }
}
