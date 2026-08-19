import XCTest
@testable import AICriticMacShared

final class ITermSwitcherFormatterTests: XCTestCase {
    func testTitleWithNote() {
        let title = ITermSwitcherFormatter.formatSessionTitle(
            name: "grok review",
            note: "fix auth on staging",
            cwd: "~/proj",
            sessionID: "sess-a"
        )
        XCTAssertEqual(title, "grok review  ·  fix auth on staging")
    }

    func testEmptyNameFallsBackToCwd() {
        let title = ITermSwitcherFormatter.formatSessionTitle(
            name: "  ",
            note: "",
            cwd: "~/proj/ai-critic",
            sessionID: "aaaa-bbbb"
        )
        XCTAssertEqual(title, "~/proj/ai-critic")
    }

    func testEmptyNameAndCwdFallsBackToShortID() {
        let title = ITermSwitcherFormatter.formatSessionTitle(
            name: "",
            note: "",
            cwd: "",
            sessionID: "D922B298-25FB-41FA-BAF8-7AC7A1D56758"
        )
        XCTAssertEqual(title, "D922B298")
    }

    func testDesktopHeaderIsOneBased() {
        XCTAssertEqual(ITermSwitcherFormatter.formatDesktopHeader(spaceIndex: 2), "Desktop 3")
    }

    func testSavedNotesHeader() {
        XCTAssertEqual(ITermSwitcherFormatter.formatSavedNotesHeader(count: 1), "Saved notes (1)")
    }

    func testSessionMatchesNote() {
        XCTAssertTrue(ITermSwitcherFormatter.sessionMatches(
            name: "grok review",
            note: "fix auth on staging",
            cwd: "",
            windowName: "",
            tabName: "",
            sessionID: "s",
            spaceIndex: 0,
            query: "auth"
        ))
    }

    func testSidebarTitles() {
        XCTAssertEqual(ITermSwitcherFormatter.formatSidebarTitle(id: ITermSwitcherFormatter.sidebarAll), "All")
        XCTAssertEqual(ITermSwitcherFormatter.formatSidebarTitle(id: ITermSwitcherFormatter.sidebarBookmarks), "Bookmarks")
        XCTAssertEqual(ITermSwitcherFormatter.formatSidebarTitle(id: ITermSwitcherFormatter.sidebarSaved), "Saved notes")
        XCTAssertEqual(ITermSwitcherFormatter.formatSidebarTitle(id: ITermSwitcherFormatter.sidebarDesktopID(spaceIndex: 2)), "Desktop 3")
        XCTAssertEqual(ITermSwitcherFormatter.formatWindowTitle(), "Terminals")
    }

    func testMatchesSidebarBookmarks() {
        XCTAssertTrue(ITermSwitcherFormatter.matchesSidebar(id: ITermSwitcherFormatter.sidebarBookmarks, spaceIndex: 0, bookmarked: true))
        XCTAssertFalse(ITermSwitcherFormatter.matchesSidebar(id: ITermSwitcherFormatter.sidebarBookmarks, spaceIndex: 0, bookmarked: false))
        XCTAssertTrue(ITermSwitcherFormatter.matchesSidebar(id: ITermSwitcherFormatter.sidebarDesktopID(spaceIndex: 1), spaceIndex: 1, bookmarked: false))
        XCTAssertFalse(ITermSwitcherFormatter.matchesSidebar(id: ITermSwitcherFormatter.sidebarDesktopID(spaceIndex: 1), spaceIndex: 0, bookmarked: true))
        XCTAssertFalse(ITermSwitcherFormatter.matchesSidebar(id: ITermSwitcherFormatter.sidebarSaved, spaceIndex: 0, bookmarked: true))
    }

    func testResolvedBookmarkedPrefersOverride() {
        XCTAssertTrue(ITermSwitcherFormatter.resolvedBookmarked(
            sessionID: "a",
            inventoryValue: false,
            overrides: ["a": true]
        ))
        XCTAssertFalse(ITermSwitcherFormatter.resolvedBookmarked(
            sessionID: "a",
            inventoryValue: true,
            overrides: ["a": false]
        ))
        XCTAssertTrue(ITermSwitcherFormatter.resolvedBookmarked(
            sessionID: "a",
            inventoryValue: true,
            overrides: [:]
        ))
    }

    func testReconcileBookmarkOverrides() {
        let got = ITermSwitcherFormatter.reconcileBookmarkOverrides(
            overrides: ["a": true, "b": true, "gone": true],
            live: ["a": true, "b": false]
        )
        XCTAssertNil(got["a"])
        XCTAssertEqual(got["b"], true)
        XCTAssertNil(got["gone"])
    }

    func testOrphanPrimaryFallsBack() {
        XCTAssertEqual(
            ITermSwitcherFormatter.formatOrphanPrimary(note: "cut last Friday", sessionName: "old", cwd: "~/d", sessionID: "id"),
            "cut last Friday"
        )
        XCTAssertEqual(
            ITermSwitcherFormatter.formatOrphanPrimary(note: "", sessionName: "old deploy", cwd: "~/d", sessionID: "id"),
            "old deploy"
        )
    }

    func testSpaceLabelRowAndSidebarTitle() {
        XCTAssertEqual(ITermSwitcherFormatter.formatSpaceLabelRow(""), "Set Space Label")
        XCTAssertEqual(ITermSwitcherFormatter.formatSpaceLabelRow("  Review staging  "), "Review staging")
        XCTAssertEqual(ITermSwitcherFormatter.formatChangeSpaceLabel(), "Change")
        XCTAssertEqual(ITermSwitcherFormatter.formatClearSpaceLabel(), "Clear")
        XCTAssertEqual(ITermSwitcherFormatter.formatShowSpaceLabel(), "Show")
        XCTAssertEqual(ITermSwitcherFormatter.formatEditSpaceLabel(), "Edit")
        XCTAssertEqual(ITermSwitcherFormatter.formatDismissSpaceLabel(), "Dismiss")
        XCTAssertEqual(ITermSwitcherFormatter.formatSidebarDesktopTitle(spaceIndex: 2, label: ""), "Desktop 3")
        XCTAssertEqual(ITermSwitcherFormatter.formatSidebarDesktopTitle(spaceIndex: 2, label: "Review staging"), "Review staging")
        XCTAssertEqual(ITermSwitcherFormatter.formatDesktopSidebarSymbol(current: false), "macwindow")
        XCTAssertEqual(ITermSwitcherFormatter.formatDesktopSidebarSymbol(current: true), "macwindow")
        XCTAssertEqual(ITermSwitcherFormatter.initialSidebarID(currentSpaceIndex: nil), ITermSwitcherFormatter.sidebarAll)
        XCTAssertEqual(ITermSwitcherFormatter.initialSidebarID(currentSpaceIndex: 12), ITermSwitcherFormatter.sidebarDesktopID(spaceIndex: 12))
    }

    func testShouldSwitchSpaceOnlyDesktops() {
        XCTAssertTrue(ITermSwitcherFormatter.shouldSwitchSpace(sidebarID: ITermSwitcherFormatter.sidebarDesktopID(spaceIndex: 0)))
        XCTAssertTrue(ITermSwitcherFormatter.shouldSwitchSpace(sidebarID: ITermSwitcherFormatter.sidebarDesktopID(spaceIndex: 12)))
        XCTAssertFalse(ITermSwitcherFormatter.shouldSwitchSpace(sidebarID: ITermSwitcherFormatter.sidebarAll))
        XCTAssertFalse(ITermSwitcherFormatter.shouldSwitchSpace(sidebarID: ITermSwitcherFormatter.sidebarBookmarks))
        XCTAssertFalse(ITermSwitcherFormatter.shouldSwitchSpace(sidebarID: ITermSwitcherFormatter.sidebarSaved))
        XCTAssertFalse(ITermSwitcherFormatter.shouldSwitchSpace(sidebarID: ""))
        XCTAssertFalse(ITermSwitcherFormatter.shouldSwitchSpace(sidebarID: "desktop:-1"))
        XCTAssertEqual(ITermSwitcherFormatter.formatSwitchSpaceMissingID(), "Can't switch Desktop — space id is missing")
        XCTAssertEqual(ITermSwitcherFormatter.formatSwitchSpaceFailed(), "Can't switch Desktop")
    }

    func testDefaultHotKey() {
        XCTAssertEqual(ITermSwitcherFormatter.formatDefaultHotKey(), "⌘⇧Space")
        XCTAssertEqual(
            ITermSwitcherFormatter.formatHotKey(
                keyCode: ITermSwitcherHotKey.defaultKeyCode,
                modifiers: ITermSwitcherHotKey.defaultModifiers
            ),
            "⌘⇧Space"
        )
    }
}
