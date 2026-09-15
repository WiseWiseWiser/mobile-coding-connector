import XCTest
@testable import AICriticMacShared

final class UsageMenuBarTests: XCTestCase {
    private func item(
        id: String,
        label: String = "Item",
        enabled: Bool = true,
        title: String? = nil,
        dropdown: String? = nil
    ) -> UsageItemView {
        UsageItemView(
            id: id,
            label: label,
            kind: "commandcode",
            enabled: enabled,
            status: "ready",
            title: title ?? "\(label) 5%",
            dropdown: dropdown ?? "\(label): 5% used, $66.19 left",
            detail: nil,
            usageURL: nil,
            error: nil,
            updatedAt: nil
        )
    }

    private func response(
        defaultID: String,
        rotate: Bool,
        items: [UsageItemView]
    ) -> UsageItemsResponse {
        UsageItemsResponse(version: 1, defaultID: defaultID, rotate: rotate, items: items)
    }

    func testDecodesServerPayload() throws {
        let json = """
        {"version":1,"default":"cc-v1","rotate":false,"items":[
          {"id":"cc-v1","label":"CC v1","kind":"commandcode","enabled":true,"home":"/tmp/cc1",
           "status":"ready","title":"CC v1 5%","dropdown":"CC v1: 5% used, $66.19 left",
           "detail":"GOAT Plan · active","usage_url":"https://commandcode.ai/acct/settings/usage",
           "updated_at":"2026-09-15T12:00:00Z"}]}
        """
        let decoded = try JSONDecoder().decode(UsageItemsResponse.self, from: Data(json.utf8))

        XCTAssertEqual(decoded.defaultID, "cc-v1")
        XCTAssertFalse(decoded.rotate)
        XCTAssertEqual(decoded.items.count, 1)
        let first = try XCTUnwrap(decoded.items.first)
        XCTAssertEqual(first.title, "CC v1 5%")
        XCTAssertEqual(first.dropdown, "CC v1: 5% used, $66.19 left")
        XCTAssertEqual(first.detail, "GOAT Plan · active")
        XCTAssertEqual(first.usageURL, "https://commandcode.ai/acct/settings/usage")
        XCTAssertEqual(first.updatedAt, "2026-09-15T12:00:00Z")
    }

    func testPinnedDefaultShowsThatItem() {
        let resp = response(defaultID: "cc-v2", rotate: false, items: [
            item(id: "grok", label: "Grok"),
            item(id: "cc-v2", label: "CommandCode cc-v2"),
        ])

        XCTAssertEqual(UsageMenuBar.title(resp, rotatingIndex: 0), "CommandCode cc-v2 5%")
        XCTAssertEqual(UsageMenuBar.title(resp, rotatingIndex: 7), "CommandCode cc-v2 5%")
    }

    func testPinnedDefaultFallsBackToFirstEnabled() {
        let resp = response(defaultID: "missing", rotate: false, items: [
            item(id: "grok", label: "Grok"),
            item(id: "cc-v1", label: "CC v1"),
        ])

        XCTAssertEqual(UsageMenuBar.title(resp, rotatingIndex: 3), "Grok 5%")
    }

    func testRotationWalksEnabledItemsInRegistryOrder() {
        let resp = response(defaultID: "grok", rotate: true, items: [
            item(id: "grok", label: "Grok"),
            item(id: "codex", label: "Codex"),
            item(id: "cc-v1", label: "CC v1"),
        ])

        XCTAssertEqual(UsageMenuBar.title(resp, rotatingIndex: 0), "Grok 5%")
        XCTAssertEqual(UsageMenuBar.title(resp, rotatingIndex: 1), "Codex 5%")
        XCTAssertEqual(UsageMenuBar.title(resp, rotatingIndex: 2), "CC v1 5%")
        XCTAssertEqual(UsageMenuBar.title(resp, rotatingIndex: 3), "Grok 5%")

        XCTAssertEqual(
            UsageMenuBar.nextRotatingIndex(resp, current: 0),
            1,
            "rotation must advance to the next item"
        )
        XCTAssertEqual(
            UsageMenuBar.nextRotatingIndex(resp, current: 2),
            0,
            "rotation must wrap around"
        )
    }

    func testRotationStartsAtDefaultItem() {
        let resp = response(defaultID: "cc-v1", rotate: true, items: [
            item(id: "grok", label: "Grok"),
            item(id: "codex", label: "Codex"),
            item(id: "cc-v1", label: "CC v1"),
        ])

        XCTAssertEqual(UsageMenuBar.title(resp, rotatingIndex: 0), "CC v1 5%")
        XCTAssertEqual(UsageMenuBar.title(resp, rotatingIndex: 1), "Grok 5%")
        XCTAssertEqual(UsageMenuBar.title(resp, rotatingIndex: 2), "Codex 5%")
        XCTAssertEqual(UsageMenuBar.title(resp, rotatingIndex: 3), "CC v1 5%")
    }

    func testRotationFallsBackToFirstEnabledWhenDefaultUnknown() {
        let resp = response(defaultID: "gone", rotate: true, items: [
            item(id: "grok", label: "Grok"),
            item(id: "cc-v1", label: "CC v1"),
        ])

        XCTAssertEqual(UsageMenuBar.title(resp, rotatingIndex: 0), "Grok 5%")
        XCTAssertEqual(UsageMenuBar.title(resp, rotatingIndex: 1), "CC v1 5%")
    }

    func testRotationSkipsDisabledItems() {
        let resp = response(defaultID: "grok", rotate: true, items: [
            item(id: "grok", label: "Grok"),
            item(id: "codex", label: "Codex", enabled: false),
            item(id: "cc-v1", label: "CC v1"),
        ])

        XCTAssertEqual(UsageMenuBar.title(resp, rotatingIndex: 1), "CC v1 5%")
        XCTAssertEqual(UsageMenuBar.nextRotatingIndex(resp, current: 1), 0)
    }

    func testDropdownLinesCoverEveryEnabledItemVerbatim() {
        let resp = response(defaultID: "grok", rotate: true, items: [
            item(id: "grok", label: "Grok", dropdown: "Grok: 12%(Weekly), Reset Sep 15, 10:00, left 2d6h"),
            item(id: "codex", label: "Codex", enabled: false, dropdown: "Codex: hidden"),
            item(
                id: "cc-v1",
                label: "CC v1",
                dropdown: "CC v1: 5% used, $66.19 left, 1,870 requests, 5h 15%, Weekly 11%, renews in 26d"
            ),
        ])

        XCTAssertEqual(UsageMenuBar.dropdownLines(resp), [
            "Grok: 12%(Weekly), Reset Sep 15, 10:00, left 2d6h",
            "CC v1: 5% used, $66.19 left, 1,870 requests, 5h 15%, Weekly 11%, renews in 26d",
        ])
    }

    func testMissingResponseKeepsFallbackTitle() {
        XCTAssertEqual(UsageMenuBar.title(nil, rotatingIndex: 0), UsageMenuBar.fallbackTitle)
        XCTAssertNil(UsageMenuBar.selectedItem(nil, rotatingIndex: 0))
        XCTAssertEqual(UsageMenuBar.dropdownLines(nil), [])
        XCTAssertEqual(UsageMenuBar.nextRotatingIndex(nil, current: 4), 0)
    }

    func testNoEnabledItemsKeepsFallbackTitle() {
        let resp = response(defaultID: "grok", rotate: true, items: [
            item(id: "grok", label: "Grok", enabled: false),
        ])

        XCTAssertEqual(UsageMenuBar.title(resp, rotatingIndex: 2), UsageMenuBar.fallbackTitle)
        XCTAssertEqual(UsageMenuBar.dropdownLines(resp), [])
        XCTAssertEqual(UsageMenuBar.nextRotatingIndex(resp, current: 2), 0)
    }

    func testErrorItemStillRendersServerText() {
        let failing = UsageItemView(
            id: "cc-v1",
            label: "CC v1",
            kind: "commandcode",
            enabled: true,
            status: "error",
            title: "CC v1 err",
            dropdown: "CC v1: Error: Session expired",
            detail: nil,
            usageURL: nil,
            error: "Session expired",
            updatedAt: "2026-09-15T12:00:00Z"
        )
        let resp = response(defaultID: "cc-v1", rotate: false, items: [failing])

        XCTAssertEqual(UsageMenuBar.title(resp, rotatingIndex: 0), "CC v1 err")
        XCTAssertEqual(UsageMenuBar.dropdownLines(resp), ["CC v1: Error: Session expired"])
    }
}
