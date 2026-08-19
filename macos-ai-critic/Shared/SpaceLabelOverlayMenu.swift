import Foundation

/// One iTerm tab in the overlay dropdown (split panes collapsed).
public struct SpaceLabelOverlayTab: Equatable {
    public var tabIndex: Int
    public var title: String
    public var sessionID: String

    public init(tabIndex: Int, title: String, sessionID: String) {
        self.tabIndex = tabIndex
        self.title = title
        self.sessionID = sessionID
    }
}

/// One iTerm window and its tabs on this Space.
public struct SpaceLabelOverlayWindow: Equatable {
    public var windowID: String
    public var title: String
    public var tabs: [SpaceLabelOverlayTab]

    public init(windowID: String, title: String, tabs: [SpaceLabelOverlayTab]) {
        self.windowID = windowID
        self.title = title
        self.tabs = tabs
    }
}

/// One row in the overlay ▾ menu (status / window header / tab / empty).
public enum SpaceLabelOverlayMenuRow: Equatable {
    case status(String)
    case empty(String)
    case window(String)
    case separator
    case tab(title: String, sessionID: String)
}

/// Group this Space's sessions into windows → tabs for the overlay menu.
public enum SpaceLabelOverlayMenu {
    public static func formatWindowTitle(_ name: String) -> String {
        let trimmed = name.trimmingCharacters(in: .whitespacesAndNewlines)
        return trimmed.isEmpty ? "Window" : trimmed
    }

    public static let markTitleLimit = 48

    public static func formatTabTitle(tabName: String, sessionName: String, cwd: String, sessionID: String, mark: String = "") -> String {
        let content = mark.trimmingCharacters(in: .whitespacesAndNewlines)
        if !content.isEmpty {
            return "mark: " + truncateMark(content)
        }
        let tab = tabName.trimmingCharacters(in: .whitespacesAndNewlines)
        if !tab.isEmpty { return tab }
        return ITermSwitcherFormatter.formatSessionPrimary(name: sessionName, cwd: cwd, sessionID: sessionID)
    }

    public static func truncateMark(_ content: String) -> String {
        if content.count <= markTitleLimit { return content }
        return String(content.prefix(markTitleLimit)) + "…"
    }

    public static func formatEmpty() -> String {
        "No iTerm tabs"
    }

    public static func formatLoading() -> String {
        "Loading tabs…"
    }

    public static func formatUpdating() -> String {
        "Updating…"
    }

    /// Suffix on the first window header while a recapture is in flight.
    public static let refreshingMark = "  ↻"

    public static func withRefreshingMark(_ title: String) -> String {
        title + refreshingMark
    }

    public static func formatMenuButton() -> String {
        "Tabs"
    }

    /// Hit box for the ▾ control (matches the switcher star).
    public static let menuButtonHitSize: CGFloat = 22

    /// Visible tab rows without the in-flight ↻ suffix.
    public static func contentRows(sessions: [ITermLiveSession]) -> [SpaceLabelOverlayMenuRow] {
        rows(sessions: sessions, refreshing: false)
    }

    /// True when the menu must be rebuilt (titles / ids / order changed).
    public static func shouldReplaceMenu(before: [ITermLiveSession], after: [ITermLiveSession]) -> Bool {
        contentRows(sessions: before) != contentRows(sessions: after)
    }

    /// Recapture in flight must not block opening the menu.
    public static func shouldPresentMenu(editing: Bool, menuRefreshing: Bool) -> Bool {
        _ = menuRefreshing
        return !editing
    }

    /// Only one recapture at a time.
    public static func shouldStartRefresh(alreadyRefreshing: Bool) -> Bool {
        !alreadyRefreshing
    }

    /// Menu rows: Loading when empty+refreshing; else grouped tabs, ↻ on first header.
    public static func rows(sessions: [ITermLiveSession], refreshing: Bool) -> [SpaceLabelOverlayMenuRow] {
        let groups = groupTabs(sessions)
        if groups.isEmpty {
            if refreshing {
                return [.status(formatLoading())]
            }
            return [.empty(formatEmpty())]
        }
        var out: [SpaceLabelOverlayMenuRow] = []
        for (i, win) in groups.enumerated() {
            if i > 0 {
                out.append(.separator)
            }
            let title = (refreshing && i == 0) ? withRefreshingMark(win.title) : win.title
            out.append(.window(title))
            for tab in win.tabs {
                out.append(.tab(title: tab.title, sessionID: tab.sessionID))
            }
        }
        return out
    }

    public static func groupTabs(_ sessions: [ITermLiveSession]) -> [SpaceLabelOverlayWindow] {
        var order: [String] = []
        var names: [String: String] = [:]
        var tabs: [String: [Int: SpaceLabelOverlayTab]] = [:]

        for sess in sessions {
            let key = sess.windowID.trimmingCharacters(in: .whitespacesAndNewlines)
            if names[key] == nil {
                order.append(key)
                names[key] = sess.windowName
                tabs[key] = [:]
            } else if names[key]?.trimmingCharacters(in: .whitespacesAndNewlines).isEmpty == true {
                let win = sess.windowName.trimmingCharacters(in: .whitespacesAndNewlines)
                if !win.isEmpty {
                    names[key] = sess.windowName
                }
            }
            if tabs[key]?[sess.tabIndex] != nil {
                continue
            }
            tabs[key]?[sess.tabIndex] = SpaceLabelOverlayTab(
                tabIndex: sess.tabIndex,
                title: formatTabTitle(
                    tabName: sess.tabName,
                    sessionName: sess.sessionName,
                    cwd: sess.cwd,
                    sessionID: sess.sessionID,
                    mark: sess.mark
                ),
                sessionID: sess.sessionID
            )
        }

        return order.map { key in
            let byIndex = tabs[key] ?? [:]
            let listed = byIndex.keys.sorted().compactMap { byIndex[$0] }
            return SpaceLabelOverlayWindow(
                windowID: key,
                title: formatWindowTitle(names[key] ?? ""),
                tabs: listed
            )
        }
    }
}
