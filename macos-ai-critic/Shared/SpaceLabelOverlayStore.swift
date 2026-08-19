import Foundation

/// One Space's overlay chrome (hide + position). Label text lives in space-labels.json.
public struct SpaceLabelOverlayItem: Codable, Equatable {
    public var spaceID: UInt64
    public var uuid: String
    public var hidden: Bool
    public var x: Double?
    public var y: Double?

    public init(spaceID: UInt64, uuid: String = "", hidden: Bool = false, x: Double? = nil, y: Double? = nil) {
        self.spaceID = spaceID
        self.uuid = uuid
        self.hidden = hidden
        self.x = x
        self.y = y
    }

    enum CodingKeys: String, CodingKey {
        case spaceID = "space_id"
        case uuid
        case hidden
        case x
        case y
    }
}

/// Versioned file at ~/.ai-critic/space-label-overlays.json.
public struct SpaceLabelOverlayDocument: Codable, Equatable {
    public var version: Int
    public var items: [SpaceLabelOverlayItem]

    public init(version: Int = SpaceLabelOverlayStore.documentVersion, items: [SpaceLabelOverlayItem] = []) {
        self.version = version
        self.items = items
    }
}

/// Live CGS identity used to rematch overlay chrome.
public struct SpaceLabelOverlayLive: Equatable {
    public var spaceID: UInt64
    public var uuid: String

    public init(spaceID: UInt64, uuid: String = "") {
        self.spaceID = spaceID
        self.uuid = uuid
    }
}

/// Load/save and pure mutations for overlay chrome.
public enum SpaceLabelOverlayStore {
    public static let documentVersion = 1

    public static func defaultPath() -> String {
        (NSHomeDirectory() as NSString).appendingPathComponent(".ai-critic/space-label-overlays.json")
    }

    public static func load(path: String) -> SpaceLabelOverlayDocument {
        guard let data = try? Data(contentsOf: URL(fileURLWithPath: path)),
              !data.isEmpty else {
            return SpaceLabelOverlayDocument()
        }
        let trimmed = String(data: data, encoding: .utf8)?
            .trimmingCharacters(in: .whitespacesAndNewlines) ?? ""
        if trimmed.isEmpty {
            return SpaceLabelOverlayDocument()
        }
        guard var doc = try? JSONDecoder().decode(SpaceLabelOverlayDocument.self, from: data) else {
            return SpaceLabelOverlayDocument()
        }
        doc.version = documentVersion
        return doc
    }

    public static func save(_ doc: SpaceLabelOverlayDocument, path: String) throws {
        let url = URL(fileURLWithPath: path)
        try FileManager.default.createDirectory(
            at: url.deletingLastPathComponent(),
            withIntermediateDirectories: true
        )
        var out = doc
        out.version = documentVersion
        let enc = JSONEncoder()
        enc.outputFormatting = [.prettyPrinted, .sortedKeys]
        let data = try enc.encode(out)
        let tmp = url.appendingPathExtension("tmp")
        try data.write(to: tmp, options: .atomic)
        _ = try? FileManager.default.removeItem(at: url)
        try FileManager.default.moveItem(at: tmp, to: url)
    }

    public static func find(_ items: [SpaceLabelOverlayItem], spaceID: UInt64, uuid: String) -> Int {
        if spaceID != 0 {
            if let i = items.firstIndex(where: { $0.spaceID == spaceID }) {
                return i
            }
        }
        let u = uuid.trimmingCharacters(in: .whitespacesAndNewlines)
        if u.isEmpty { return -1 }
        return items.firstIndex(where: { $0.uuid.trimmingCharacters(in: .whitespacesAndNewlines) == u }) ?? -1
    }

    public static func item(_ items: [SpaceLabelOverlayItem], spaceID: UInt64, uuid: String) -> SpaceLabelOverlayItem? {
        let i = find(items, spaceID: spaceID, uuid: uuid)
        return i >= 0 ? items[i] : nil
    }

    public static func isHidden(_ items: [SpaceLabelOverlayItem], spaceID: UInt64, uuid: String) -> Bool {
        item(items, spaceID: spaceID, uuid: uuid)?.hidden ?? false
    }

    public static func hiddenIDs(_ items: [SpaceLabelOverlayItem]) -> Set<UInt64> {
        Set(items.filter { $0.hidden && $0.spaceID != 0 }.map(\.spaceID))
    }

    /// Keep/relink chrome for live Spaces; drop rows whose Space is gone.
    public static func rematch(
        _ items: [SpaceLabelOverlayItem],
        live: [SpaceLabelOverlayLive]
    ) -> (items: [SpaceLabelOverlayItem], changed: Bool) {
        var byID: [UInt64: SpaceLabelOverlayLive] = [:]
        var byUUID: [String: SpaceLabelOverlayLive] = [:]
        for s in live {
            if s.spaceID != 0 { byID[s.spaceID] = s }
            let u = s.uuid.trimmingCharacters(in: .whitespacesAndNewlines)
            if !u.isEmpty { byUUID[u] = s }
        }
        var out: [SpaceLabelOverlayItem] = []
        var changed = false
        for it in items {
            if it.spaceID != 0, let hit = byID[it.spaceID] {
                var cp = it
                let u = hit.uuid.trimmingCharacters(in: .whitespacesAndNewlines)
                if !u.isEmpty && cp.uuid != u {
                    cp.uuid = u
                    changed = true
                }
                out.append(cp)
                continue
            }
            let u = it.uuid.trimmingCharacters(in: .whitespacesAndNewlines)
            if !u.isEmpty, let hit = byUUID[u] {
                var cp = it
                if cp.spaceID != hit.spaceID {
                    cp.spaceID = hit.spaceID
                    changed = true
                }
                out.append(cp)
                continue
            }
            changed = true
        }
        if !changed && out.count == items.count {
            return (items, false)
        }
        return (out, true)
    }

    public static func setHidden(
        _ items: [SpaceLabelOverlayItem],
        spaceID: UInt64,
        uuid: String,
        hidden: Bool
    ) -> [SpaceLabelOverlayItem] {
        upsert(items, spaceID: spaceID, uuid: uuid) { $0.hidden = hidden }
    }

    public static func setPosition(
        _ items: [SpaceLabelOverlayItem],
        spaceID: UInt64,
        uuid: String,
        x: Double,
        y: Double
    ) -> [SpaceLabelOverlayItem] {
        upsert(items, spaceID: spaceID, uuid: uuid) {
            $0.x = x
            $0.y = y
        }
    }

    public static func upsert(
        _ items: [SpaceLabelOverlayItem],
        spaceID: UInt64,
        uuid: String,
        mutate: (inout SpaceLabelOverlayItem) -> Void
    ) -> [SpaceLabelOverlayItem] {
        var out = items
        let i = find(out, spaceID: spaceID, uuid: uuid)
        if i >= 0 {
            var rec = out[i]
            if rec.spaceID == 0 { rec.spaceID = spaceID }
            if rec.uuid.trimmingCharacters(in: .whitespacesAndNewlines).isEmpty {
                rec.uuid = uuid.trimmingCharacters(in: .whitespacesAndNewlines)
            }
            mutate(&rec)
            out[i] = rec
            return out
        }
        var rec = SpaceLabelOverlayItem(
            spaceID: spaceID,
            uuid: uuid.trimmingCharacters(in: .whitespacesAndNewlines)
        )
        mutate(&rec)
        out.append(rec)
        return out
    }
}
