import XCTest
@testable import AICriticMacShared

final class SpaceLabelOverlayMenuTests: XCTestCase {
    func testEmptySessions() {
        XCTAssertEqual(SpaceLabelOverlayMenu.groupTabs([]), [])
        XCTAssertEqual(SpaceLabelOverlayMenu.formatEmpty(), "No iTerm tabs")
        XCTAssertEqual(SpaceLabelOverlayMenu.formatLoading(), "Loading tabs…")
        XCTAssertEqual(SpaceLabelOverlayMenu.formatUpdating(), "Updating…")
        XCTAssertEqual(SpaceLabelOverlayMenu.formatMenuButton(), "Tabs")
        XCTAssertEqual(SpaceLabelOverlayMenu.menuButtonHitSize, 22)
        XCTAssertTrue(SpaceLabelOverlayMenu.shouldPresentMenu(editing: false, menuRefreshing: true))
        XCTAssertTrue(SpaceLabelOverlayMenu.shouldPresentMenu(editing: false, menuRefreshing: false))
        XCTAssertFalse(SpaceLabelOverlayMenu.shouldPresentMenu(editing: true, menuRefreshing: false))
        XCTAssertTrue(SpaceLabelOverlayMenu.shouldStartRefresh(alreadyRefreshing: false))
        XCTAssertFalse(SpaceLabelOverlayMenu.shouldStartRefresh(alreadyRefreshing: true))
        XCTAssertEqual(
            SpaceLabelOverlayMenu.rows(sessions: [], refreshing: false),
            [.empty("No iTerm tabs")]
        )
        XCTAssertEqual(
            SpaceLabelOverlayMenu.rows(sessions: [], refreshing: true),
            [.status("Loading tabs…")]
        )
    }

    func testRowsRefreshingKeepsCachedTabs() {
        let sessions = [
            ITermLiveSession(sessionID: "s1", sessionName: "zsh", windowID: "w1", windowName: "work", tabIndex: 1, tabName: "Local"),
            ITermLiveSession(sessionID: "s2", sessionName: "grok", windowID: "w2", windowName: "other", tabIndex: 1, tabName: "grok"),
        ]
        XCTAssertEqual(
            SpaceLabelOverlayMenu.rows(sessions: sessions, refreshing: true),
            [
                .window(SpaceLabelOverlayMenu.withRefreshingMark("work")),
                .tab(title: "Local", sessionID: "s1"),
                .separator,
                .window("other"),
                .tab(title: "grok", sessionID: "s2"),
            ]
        )
        let idle = SpaceLabelOverlayMenu.rows(sessions: sessions, refreshing: false)
        XCTAssertEqual(
            idle,
            [
                .window("work"),
                .tab(title: "Local", sessionID: "s1"),
                .separator,
                .window("other"),
                .tab(title: "grok", sessionID: "s2"),
            ]
        )
        XCTAssertEqual(SpaceLabelOverlayMenu.contentRows(sessions: sessions), idle)
        XCTAssertFalse(SpaceLabelOverlayMenu.shouldReplaceMenu(before: sessions, after: sessions))
        let renamed = [
            ITermLiveSession(sessionID: "s1", sessionName: "zsh", windowID: "w1", windowName: "work", tabIndex: 1, tabName: "Local"),
            ITermLiveSession(sessionID: "s2", sessionName: "grok", windowID: "w2", windowName: "other", tabIndex: 1, tabName: "mark", mark: "persist bookmark"),
        ]
        XCTAssertTrue(SpaceLabelOverlayMenu.shouldReplaceMenu(before: sessions, after: renamed))
    }

    func testGroupsTabsByWindow() {
        let sessions = [
            ITermLiveSession(sessionID: "s1", sessionName: "zsh", windowID: "w1", windowName: "work", tabIndex: 1, tabName: "Local"),
            ITermLiveSession(sessionID: "s2", sessionName: "grok", windowID: "w1", windowName: "work", tabIndex: 2, tabName: "grok"),
            ITermLiveSession(sessionID: "s3", sessionName: "t", windowID: "w2", windowName: "ai-critic", tabIndex: 1, tabName: "Terminal"),
        ]
        let got = SpaceLabelOverlayMenu.groupTabs(sessions)
        XCTAssertEqual(got.count, 2)
        XCTAssertEqual(got[0].windowID, "w1")
        XCTAssertEqual(got[0].title, "work")
        XCTAssertEqual(got[0].tabs.map(\.title), ["Local", "grok"])
        XCTAssertEqual(got[0].tabs.map(\.sessionID), ["s1", "s2"])
        XCTAssertEqual(got[1].windowID, "w2")
        XCTAssertEqual(got[1].title, "ai-critic")
        XCTAssertEqual(got[1].tabs.map(\.sessionID), ["s3"])
    }

    func testSplitPanesCollapseToFirstSession() {
        let sessions = [
            ITermLiveSession(sessionID: "left", sessionName: "left", windowID: "w", windowName: "win", tabIndex: 3, tabName: "split"),
            ITermLiveSession(sessionID: "right", sessionName: "right", windowID: "w", windowName: "win", tabIndex: 3, tabName: "split"),
        ]
        let got = SpaceLabelOverlayMenu.groupTabs(sessions)
        XCTAssertEqual(got.count, 1)
        XCTAssertEqual(got[0].tabs.count, 1)
        XCTAssertEqual(got[0].tabs[0].sessionID, "left")
        XCTAssertEqual(got[0].tabs[0].title, "split")
    }

    func testTabsSortedByIndex() {
        let sessions = [
            ITermLiveSession(sessionID: "b", windowID: "w", windowName: "win", tabIndex: 4, tabName: "later"),
            ITermLiveSession(sessionID: "a", windowID: "w", windowName: "win", tabIndex: 1, tabName: "first"),
        ]
        let got = SpaceLabelOverlayMenu.groupTabs(sessions)
        XCTAssertEqual(got[0].tabs.map(\.title), ["first", "later"])
    }

    func testWindowTitleFallback() {
        XCTAssertEqual(SpaceLabelOverlayMenu.formatWindowTitle(""), "Window")
        XCTAssertEqual(SpaceLabelOverlayMenu.formatWindowTitle("  "), "Window")
        XCTAssertEqual(SpaceLabelOverlayMenu.formatWindowTitle(" work "), "work")
        let got = SpaceLabelOverlayMenu.groupTabs([
            ITermLiveSession(sessionID: "s", windowID: "w", windowName: "  ", tabIndex: 0, tabName: "t"),
        ])
        XCTAssertEqual(got[0].title, "Window")
    }

    func testTabTitleFallbacks() {
        XCTAssertEqual(
            SpaceLabelOverlayMenu.formatTabTitle(tabName: " Local ", sessionName: "zsh", cwd: "~/p", sessionID: "aaaa-bbbb"),
            "Local"
        )
        XCTAssertEqual(
            SpaceLabelOverlayMenu.formatTabTitle(tabName: "  ", sessionName: "zsh", cwd: "~/p", sessionID: "aaaa-bbbb"),
            "zsh"
        )
        XCTAssertEqual(
            SpaceLabelOverlayMenu.formatTabTitle(tabName: "", sessionName: "", cwd: "~/proj", sessionID: "aaaa-bbbb"),
            "~/proj"
        )
        XCTAssertEqual(
            SpaceLabelOverlayMenu.formatTabTitle(tabName: "", sessionName: "", cwd: "", sessionID: "D922B298-25FB-41FA-BAF8-7AC7A1D56758"),
            "D922B298"
        )
    }

    func testMarkTabShowsContent() {
        XCTAssertEqual(
            SpaceLabelOverlayMenu.formatTabTitle(tabName: "mark", sessionName: "mark", cwd: "", sessionID: "s", mark: "waiting for CI"),
            "mark: waiting for CI"
        )
    }

    func testProfileMarkTitleShowsContent() {
        XCTAssertEqual(
            SpaceLabelOverlayMenu.formatTabTitle(tabName: "Default (mark)", sessionName: "Default (mark)", cwd: "", sessionID: "s", mark: "persist bookmark"),
            "mark: persist bookmark"
        )
    }

    func testMarkTabEmptyContentStaysTabName() {
        XCTAssertEqual(
            SpaceLabelOverlayMenu.formatTabTitle(tabName: "Default (mark)", sessionName: "Default (mark)", cwd: "", sessionID: "s", mark: ""),
            "Default (mark)"
        )
        XCTAssertEqual(
            SpaceLabelOverlayMenu.formatTabTitle(tabName: "mark", sessionName: "mark", cwd: "", sessionID: "s", mark: "   "),
            "mark"
        )
    }

    func testMarkContentCappedWithEllipsis() {
        let long = String(repeating: "a", count: 60)
        let got = SpaceLabelOverlayMenu.formatTabTitle(tabName: "mark", sessionName: "mark", cwd: "", sessionID: "s", mark: long)
        XCTAssertTrue(got.hasPrefix("mark: "))
        XCTAssertTrue(got.hasSuffix("…"))
        let body = String(got.dropFirst("mark: ".count).dropLast())
        XCTAssertEqual(body, String(repeating: "a", count: SpaceLabelOverlayMenu.markTitleLimit))
    }

    func testGroupTabsUsesMarkContent() {
        let sessions = [
            ITermLiveSession(sessionID: "s1", sessionName: "mark", windowID: "w", windowName: "win", tabIndex: 1, tabName: "mark", mark: "waiting for CI"),
        ]
        let got = SpaceLabelOverlayMenu.groupTabs(sessions)
        XCTAssertEqual(got[0].tabs.map(\.title), ["mark: waiting for CI"])
    }

    func testPreservesWindowOrder() {
        let sessions = [
            ITermLiveSession(sessionID: "s2", windowID: "w2", windowName: "second", tabIndex: 0, tabName: "b"),
            ITermLiveSession(sessionID: "s1", windowID: "w1", windowName: "first", tabIndex: 0, tabName: "a"),
        ]
        let got = SpaceLabelOverlayMenu.groupTabs(sessions)
        XCTAssertEqual(got.map(\.title), ["second", "first"])
    }
}
