import XCTest
@testable import AICriticMacShared

/// Proves error service titles include the full service name (not ellipsis-only).
final class ServiceMenuTitleTests: XCTestCase {
    func testErrorTitle_includesFullServiceName() {
        let title = ServiceMenuFormatter.formatServiceTitle(
            name: "web",
            status: "error",
            enabled: true
        )
        XCTAssertEqual(title, "web ⚠ Error")
        XCTAssertTrue(title.contains("web"))
        XCTAssertTrue(title.contains("⚠ Error"))
        XCTAssertNotEqual(title, "… ⚠ Error")
    }

    func testErrorTitle_longNameNotTruncated() {
        let title = ServiceMenuFormatter.formatServiceTitle(
            name: "my-long-service",
            status: "error",
            enabled: false
        )
        XCTAssertEqual(title, "my-long-service ⚠ Error")
    }

    func testRunningTitle_unchanged() {
        let title = ServiceMenuFormatter.formatServiceTitle(
            name: "web",
            status: "running",
            enabled: true
        )
        XCTAssertEqual(title, "web ● Running")
    }
}
