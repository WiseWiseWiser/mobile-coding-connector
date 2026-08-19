import XCTest
@testable import AICriticMacShared

final class MainWindowFormatterTests: XCTestCase {
    func testSidebarTitles() {
        XCTAssertEqual(MainWindowFormatter.formatSidebarTitle(id: "home"), "Home")
        XCTAssertEqual(MainWindowFormatter.formatSidebarTitle(id: "services"), "Services")
        XCTAssertEqual(MainWindowFormatter.formatSidebarTitle(id: "projects"), "Projects")
        XCTAssertEqual(MainWindowFormatter.formatSidebarTitle(id: "settings"), "Settings")
        XCTAssertEqual(MainSidebarItem.home.title, "Home")
    }

    func testShowAppLabel() {
        XCTAssertEqual(MainWindowFormatter.formatShowAppLabel(), "Show App")
    }

    func testNormalizeUnknownDefaultsToHome() {
        XCTAssertEqual(MainWindowFormatter.normalizeSidebarID("nope"), "home")
        XCTAssertEqual(MainWindowFormatter.normalizeSidebarID("projects"), "projects")
    }
}
