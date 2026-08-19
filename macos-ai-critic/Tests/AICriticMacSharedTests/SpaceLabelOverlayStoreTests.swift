import XCTest
@testable import AICriticMacShared

final class SpaceLabelOverlayStoreTests: XCTestCase {
    func testMissingFileIsEmpty() {
        let doc = SpaceLabelOverlayStore.load(path: "/tmp/does-not-exist-space-label-overlays.json")
        XCTAssertEqual(doc.items.count, 0)
        XCTAssertEqual(doc.version, SpaceLabelOverlayStore.documentVersion)
    }

    func testCorruptFileIsEmpty() {
        let dir = NSTemporaryDirectory()
        let path = (dir as NSString).appendingPathComponent("space-label-overlays-corrupt-\(UUID().uuidString).json")
        try! "{".write(toFile: path, atomically: true, encoding: .utf8)
        defer { try? FileManager.default.removeItem(atPath: path) }
        let doc = SpaceLabelOverlayStore.load(path: path)
        XCTAssertEqual(doc.items.count, 0)
    }

    func testHidePersistsWithoutClearingPosition() throws {
        let path = tempPath()
        defer { try? FileManager.default.removeItem(atPath: path) }
        var items: [SpaceLabelOverlayItem] = []
        items = SpaceLabelOverlayStore.setPosition(items, spaceID: 10, uuid: "u0", x: 0.25, y: 0.8)
        items = SpaceLabelOverlayStore.setHidden(items, spaceID: 10, uuid: "u0", hidden: true)
        try SpaceLabelOverlayStore.save(SpaceLabelOverlayDocument(items: items), path: path)
        let loaded = SpaceLabelOverlayStore.load(path: path)
        XCTAssertEqual(loaded.items.count, 1)
        XCTAssertTrue(loaded.items[0].hidden)
        XCTAssertEqual(loaded.items[0].x, 0.25)
        XCTAssertEqual(loaded.items[0].y, 0.8)
        XCTAssertEqual(loaded.items[0].spaceID, 10)
        XCTAssertTrue(SpaceLabelOverlayStore.isHidden(loaded.items, spaceID: 10, uuid: "u0"))
    }

    func testRevealClearsHidden() {
        var items = SpaceLabelOverlayStore.setHidden([], spaceID: 3, uuid: "abc", hidden: true)
        XCTAssertTrue(SpaceLabelOverlayStore.isHidden(items, spaceID: 3, uuid: "abc"))
        items = SpaceLabelOverlayStore.setHidden(items, spaceID: 3, uuid: "abc", hidden: false)
        XCTAssertFalse(SpaceLabelOverlayStore.isHidden(items, spaceID: 3, uuid: "abc"))
        XCTAssertEqual(SpaceLabelOverlayStore.hiddenIDs(items).count, 0)
    }

    func testUnknownSpaceIsNotHidden() {
        XCTAssertFalse(SpaceLabelOverlayStore.isHidden([], spaceID: 1, uuid: "x"))
    }

    func testRematchUpdatesSpaceIDKeepsChrome() {
        let stored = [
            SpaceLabelOverlayItem(spaceID: 99, uuid: "ccc", hidden: true, x: 0.2, y: 0.8),
        ]
        let live = [SpaceLabelOverlayLive(spaceID: 7, uuid: "ccc")]
        let got = SpaceLabelOverlayStore.rematch(stored, live: live)
        XCTAssertTrue(got.changed)
        XCTAssertEqual(got.items.count, 1)
        XCTAssertEqual(got.items[0].spaceID, 7)
        XCTAssertEqual(got.items[0].uuid, "ccc")
        XCTAssertTrue(got.items[0].hidden)
        XCTAssertEqual(got.items[0].x, 0.2)
        XCTAssertEqual(got.items[0].y, 0.8)
    }

    func testRematchUpdatesUUIDOnIDHit() {
        let stored = [SpaceLabelOverlayItem(spaceID: 1, uuid: "aaa")]
        let live = [SpaceLabelOverlayLive(spaceID: 1, uuid: "zzz")]
        let got = SpaceLabelOverlayStore.rematch(stored, live: live)
        XCTAssertTrue(got.changed)
        XCTAssertEqual(got.items[0].spaceID, 1)
        XCTAssertEqual(got.items[0].uuid, "zzz")
    }

    func testRematchDropsMissingSpace() {
        let stored = [
            SpaceLabelOverlayItem(spaceID: 1, uuid: "aaa"),
            SpaceLabelOverlayItem(spaceID: 2, uuid: "bbb", hidden: true, x: 0.1, y: 0.2),
        ]
        let live = [SpaceLabelOverlayLive(spaceID: 1, uuid: "aaa")]
        let got = SpaceLabelOverlayStore.rematch(stored, live: live)
        XCTAssertTrue(got.changed)
        XCTAssertEqual(got.items.count, 1)
        XCTAssertEqual(got.items[0].spaceID, 1)
    }

    func testRematchUnchangedWhenSame() {
        let stored = [SpaceLabelOverlayItem(spaceID: 1, uuid: "aaa", hidden: false, x: 0.5, y: 1)]
        let live = [SpaceLabelOverlayLive(spaceID: 1, uuid: "aaa")]
        let got = SpaceLabelOverlayStore.rematch(stored, live: live)
        XCTAssertFalse(got.changed)
        XCTAssertEqual(got.items, stored)
    }

    func testFindPrefersSpaceID() {
        let items = [
            SpaceLabelOverlayItem(spaceID: 1, uuid: "other"),
            SpaceLabelOverlayItem(spaceID: 2, uuid: "u"),
        ]
        XCTAssertEqual(SpaceLabelOverlayStore.find(items, spaceID: 1, uuid: "u"), 0)
        XCTAssertEqual(SpaceLabelOverlayStore.find(items, spaceID: 0, uuid: "u"), 1)
    }

    private func tempPath() -> String {
        let dir = NSTemporaryDirectory()
        return (dir as NSString).appendingPathComponent("space-label-overlays-\(UUID().uuidString).json")
    }
}

final class SpaceLabelOverlayLayoutTests: XCTestCase {
    func testDefaultOriginIsTopCenter() {
        let visible = CGRect(x: 100, y: 50, width: 1000, height: 800)
        let size = CGSize(width: 100, height: 30)
        let origin = SpaceLabelOverlayLayout.defaultOrigin(size: size, visible: visible)
        XCTAssertEqual(origin.x, 100 + 500 - 50, accuracy: 0.01)
        XCTAssertEqual(origin.y, 50 + 800 - 30 - 10, accuracy: 0.01)
    }

    func testNilCoordsUseDefault() {
        let visible = CGRect(x: 0, y: 0, width: 800, height: 600)
        let size = CGSize(width: 40, height: 20)
        let origin = SpaceLabelOverlayLayout.origin(x: nil, y: nil, size: size, visible: visible)
        XCTAssertEqual(origin, SpaceLabelOverlayLayout.defaultOrigin(size: size, visible: visible))
    }

    func testNormalizeRestoreRoundTrip() {
        let visible = CGRect(x: 100, y: 50, width: 1000, height: 800)
        let size = CGSize(width: 120, height: 28)
        let points = [
            CGPoint(x: 100, y: 50),
            CGPoint(x: 100 + 1000 - 120, y: 50 + 800 - 28),
            CGPoint(x: 400, y: 300),
        ]
        for origin in points {
            let n = SpaceLabelOverlayLayout.normalize(origin: origin, size: size, visible: visible)
            let back = SpaceLabelOverlayLayout.restore(x: n.x, y: n.y, size: size, visible: visible)
            XCTAssertEqual(back.x, origin.x, accuracy: 0.5, "x \(origin)")
            XCTAssertEqual(back.y, origin.y, accuracy: 0.5, "y \(origin)")
        }
    }

    func testRestoreClampsToSmallerScreen() {
        let small = CGRect(x: 0, y: 0, width: 400, height: 300)
        let size = CGSize(width: 80, height: 24)
        let origin = SpaceLabelOverlayLayout.restore(x: 1.4, y: -0.2, size: size, visible: small)
        XCTAssertGreaterThanOrEqual(origin.x, small.minX)
        XCTAssertGreaterThanOrEqual(origin.y, small.minY)
        XCTAssertLessThanOrEqual(origin.x + size.width, small.maxX + 0.01)
        XCTAssertLessThanOrEqual(origin.y + size.height, small.maxY + 0.01)
        XCTAssertEqual(origin.x, small.maxX - size.width, accuracy: 0.01)
        XCTAssertEqual(origin.y, small.minY, accuracy: 0.01)
    }

    func testClamp01() {
        XCTAssertEqual(SpaceLabelOverlayLayout.clamp01(-1), 0)
        XCTAssertEqual(SpaceLabelOverlayLayout.clamp01(2), 1)
        XCTAssertEqual(SpaceLabelOverlayLayout.clamp01(0.3), 0.3)
    }
}
