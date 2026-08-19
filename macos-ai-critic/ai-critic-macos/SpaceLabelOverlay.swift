import AppKit
import SwiftUI
import AICriticMacShared

/// Per-Space sticky pills. Click-through is off only on the capsule.
@MainActor
final class SpaceLabelOverlayController: ObservableObject {
    static let shared = SpaceLabelOverlayController()

    @Published private(set) var hiddenIDs: Set<UInt64> = []

    private var panels: [UInt64: SpaceLabelPanel] = [:]
    private var started = false
    private var refreshTask: Task<Void, Never>?
    private var storePath: String = SpaceLabelOverlayStore.defaultPath()
    private var doc = SpaceLabelOverlayDocument()

    func start() {
        guard !started else { return }
        started = true
        doc = SpaceLabelOverlayStore.load(path: storePath)
        hiddenIDs = SpaceLabelOverlayStore.hiddenIDs(doc.items)
        NSWorkspace.shared.notificationCenter.addObserver(
            self,
            selector: #selector(spaceChanged),
            name: NSWorkspace.activeSpaceDidChangeNotification,
            object: nil
        )
        refreshTask = Task { [weak self] in
            while !Task.isCancelled {
                await self?.refresh()
                try? await Task.sleep(nanoseconds: 10_000_000_000)
            }
        }
    }

    func isHidden(spaceID: UInt64, uuid: String) -> Bool {
        SpaceLabelOverlayStore.isHidden(doc.items, spaceID: spaceID, uuid: uuid)
    }

    func reveal(spaceID: UInt64, uuid: String) {
        guard spaceID != 0 || !uuid.trimmingCharacters(in: .whitespacesAndNewlines).isEmpty else { return }
        doc.items = SpaceLabelOverlayStore.setHidden(doc.items, spaceID: spaceID, uuid: uuid, hidden: false)
        persist()
    }

    func apply(desktops: [ITermDesktopGroup]) {
        rematch(desktops)
        var want: Set<UInt64> = []
        for desk in desktops {
            let text = desk.label.trimmingCharacters(in: .whitespacesAndNewlines)
            guard desk.spaceID != 0, !text.isEmpty else { continue }
            if isHidden(spaceID: desk.spaceID, uuid: desk.spaceUUID) {
                if let existing = panels[desk.spaceID] {
                    existing.close()
                    panels.removeValue(forKey: desk.spaceID)
                }
                continue
            }
            want.insert(desk.spaceID)
            if let existing = panels[desk.spaceID] {
                existing.update(text: text, uuid: desk.spaceUUID, sessions: desk.sessions)
                existing.onRefreshSpace = { [weak self] spaceID in
                    await self?.refreshSpace(spaceID: spaceID)
                }
                existing.pin(toSpaceID: desk.spaceID)
                continue
            }
            let chrome = SpaceLabelOverlayStore.item(doc.items, spaceID: desk.spaceID, uuid: desk.spaceUUID)
            let panel = SpaceLabelPanel(
                text: text,
                spaceID: desk.spaceID,
                uuid: desk.spaceUUID,
                sessions: desk.sessions,
                x: chrome?.x,
                y: chrome?.y
            )
            panel.onSave = { [weak self] next in
                Task { await self?.saveLabel(spaceID: desk.spaceID, uuid: desk.spaceUUID, label: next) }
            }
            panel.onDismiss = { [weak self] in
                self?.dismiss(spaceID: desk.spaceID, uuid: desk.spaceUUID)
            }
            panel.onMoved = { [weak self] origin, size, visible in
                self?.persistPosition(spaceID: desk.spaceID, uuid: desk.spaceUUID, origin: origin, size: size, visible: visible)
            }
            panel.onFocus = { sessionID in
                Task {
                    try? await ServerClient.shared.focusITermSession(sessionID: sessionID)
                }
            }
            panel.onRefreshSpace = { [weak self] spaceID in
                await self?.refreshSpace(spaceID: spaceID)
            }
            panel.pin(toSpaceID: desk.spaceID)
            panels[desk.spaceID] = panel
        }
        for (id, panel) in panels where !want.contains(id) {
            panel.close()
            panels.removeValue(forKey: id)
        }
    }

    @objc private func spaceChanged() {
        Task { await refresh() }
    }

    private func refresh() async {
        do {
            let inv = try await ServerClient.shared.itermInventory()
            apply(desktops: inv.desktops)
        } catch {
            // Daemon may not be up yet; retry on the next tick.
        }
    }

    /// Recapture one Desktop's tabs (marks, titles) and return its sessions.
    /// Nil means the request failed; the pill keeps the last list.
    func refreshSpace(spaceID: UInt64) async -> [ITermLiveSession]? {
        do {
            let inv = try await ServerClient.shared.itermInventory(refresh: true, spaceID: spaceID)
            apply(desktops: inv.desktops)
            return inv.desktops.first(where: { $0.spaceID == spaceID })?.sessions ?? []
        } catch {
            return nil
        }
    }

    private func rematch(_ desktops: [ITermDesktopGroup]) {
        let live = desktops.compactMap { desk -> SpaceLabelOverlayLive? in
            guard desk.spaceID != 0 else { return nil }
            return SpaceLabelOverlayLive(spaceID: desk.spaceID, uuid: desk.spaceUUID)
        }
        let next = SpaceLabelOverlayStore.rematch(doc.items, live: live)
        if next.changed {
            doc.items = next.items
            persist()
        } else {
            hiddenIDs = SpaceLabelOverlayStore.hiddenIDs(doc.items)
        }
    }

    private func dismiss(spaceID: UInt64, uuid: String) {
        doc.items = SpaceLabelOverlayStore.setHidden(doc.items, spaceID: spaceID, uuid: uuid, hidden: true)
        persist()
        if let panel = panels[spaceID] {
            panel.close()
            panels.removeValue(forKey: spaceID)
        }
    }

    private func persistPosition(
        spaceID: UInt64,
        uuid: String,
        origin: CGPoint,
        size: CGSize,
        visible: CGRect
    ) {
        let n = SpaceLabelOverlayLayout.normalize(origin: origin, size: size, visible: visible)
        doc.items = SpaceLabelOverlayStore.setPosition(doc.items, spaceID: spaceID, uuid: uuid, x: n.x, y: n.y)
        persist()
    }

    private func saveLabel(spaceID: UInt64, uuid: String, label: String) async {
        let trimmed = label.trimmingCharacters(in: .whitespacesAndNewlines)
        if !trimmed.isEmpty {
            reveal(spaceID: spaceID, uuid: uuid)
        }
        do {
            try await ServerClient.shared.setITermSpaceLabel(spaceID: spaceID, uuid: uuid, label: label)
            await refresh()
        } catch {
            // Keep the last painted pill; user can retry.
        }
    }

    private func persist() {
        hiddenIDs = SpaceLabelOverlayStore.hiddenIDs(doc.items)
        try? SpaceLabelOverlayStore.save(doc, path: storePath)
    }
}

@MainActor
private final class SpaceLabelNSPanel: NSPanel {
    override var canBecomeKey: Bool { true }
    override var canBecomeMain: Bool { false }
}

@MainActor
private final class SpaceLabelPanel: NSObject {
    private let panel: SpaceLabelNSPanel
    private let host: NSHostingView<SpaceLabelPill>
    private var text: String
    private var uuid: String
    private let spaceID: UInt64
    private var sessions: [ITermLiveSession]
    private var editing = false
    private var draft = ""
    private var applyingFrame = false
    private var persistWork: DispatchWorkItem?

    var onSave: ((String) -> Void)?
    var onDismiss: (() -> Void)?
    var onMoved: ((CGPoint, CGSize, CGRect) -> Void)?
    var onFocus: ((String) -> Void)?
    var onRefreshSpace: ((UInt64) async -> [ITermLiveSession]?)?

    private var menuRefreshing = false
    private var openMenu: NSMenu?

    init(text: String, spaceID: UInt64, uuid: String, sessions: [ITermLiveSession], x: Double?, y: Double?) {
        self.text = text
        self.spaceID = spaceID
        self.uuid = uuid
        self.sessions = sessions
        let view = SpaceLabelPill(
            text: text,
            editing: false,
            draft: "",
            onDraft: { _ in },
            onCommit: {},
            onCancel: {},
            onBeginEdit: {},
            onDismiss: {},
            onOpenMenu: {}
        )
        let host = NSHostingView(rootView: view)
        host.frame = NSRect(origin: .zero, size: host.fittingSize)
        self.host = host

        let panel = SpaceLabelNSPanel(
            contentRect: host.frame,
            styleMask: [.borderless, .nonactivatingPanel],
            backing: .buffered,
            defer: false
        )
        panel.isFloatingPanel = true
        panel.level = NSWindow.Level(rawValue: NSWindow.Level.floating.rawValue)
        panel.hidesOnDeactivate = false
        panel.isOpaque = false
        panel.backgroundColor = .clear
        panel.hasShadow = true
        panel.ignoresMouseEvents = false
        panel.becomesKeyOnlyIfNeeded = true
        panel.collectionBehavior = [.ignoresCycle, .fullScreenAuxiliary, .stationary]
        panel.titleVisibility = .hidden
        panel.titlebarAppearsTransparent = true
        panel.isMovableByWindowBackground = true
        panel.isReleasedWhenClosed = false
        panel.contentView = host
        self.panel = panel
        super.init()
        NotificationCenter.default.addObserver(
            self,
            selector: #selector(windowDidMove),
            name: NSWindow.didMoveNotification,
            object: panel
        )
        NotificationCenter.default.addObserver(
            self,
            selector: #selector(windowDidResignKey),
            name: NSWindow.didResignKeyNotification,
            object: panel
        )
        refreshView()
        place(x: x, y: y)
        panel.orderFrontRegardless()
    }

    func update(text: String, uuid: String, sessions: [ITermLiveSession]) {
        if !uuid.trimmingCharacters(in: .whitespacesAndNewlines).isEmpty {
            self.uuid = uuid
        }
        self.sessions = sessions
        guard text != self.text else { return }
        self.text = text
        if editing { return }
        refreshView()
        resizeInPlace()
    }

    func pin(toSpaceID spaceID: UInt64) {
        CGSSpaceMove.moveWindow(panel, toSpaceID: spaceID)
        if !panel.isVisible {
            panel.orderFrontRegardless()
        }
    }

    func close() {
        persistWork?.cancel()
        persistWork = nil
        openMenu = nil
        NotificationCenter.default.removeObserver(self)
        panel.orderOut(nil)
    }

    private func beginEdit() {
        guard !editing else { return }
        editing = true
        draft = text
        panel.isMovableByWindowBackground = false
        refreshView()
        panel.makeKeyAndOrderFront(nil)
    }

    private func cancelEdit() {
        guard editing else { return }
        editing = false
        draft = ""
        panel.isMovableByWindowBackground = true
        refreshView()
        resizeInPlace()
    }

    private func commitEdit() {
        guard editing else { return }
        let next = draft
        editing = false
        draft = ""
        panel.isMovableByWindowBackground = true
        refreshView()
        onSave?(next)
    }

    private func refreshView() {
        host.rootView = SpaceLabelPill(
            text: text,
            editing: editing,
            draft: draft,
            onDraft: { [weak self] value in
                self?.draft = value
            },
            onCommit: { [weak self] in
                self?.commitEdit()
            },
            onCancel: { [weak self] in
                self?.cancelEdit()
            },
            onBeginEdit: { [weak self] in
                self?.beginEdit()
            },
            onDismiss: { [weak self] in
                self?.onDismiss?()
            },
            onOpenMenu: { [weak self] in
                self?.showMenu()
            }
        )
        if !editing {
            resizeInPlace()
        }
    }

    private func resizeInPlace() {
        let size = host.fittingSize
        var frame = panel.frame
        if frame.size == size { return }
        frame.size = size
        applyingFrame = true
        panel.setFrame(frame, display: true)
        applyingFrame = false
        host.frame = NSRect(origin: .zero, size: size)
    }

    private func place(x: Double?, y: Double?) {
        let screen = panel.screen ?? NSScreen.main ?? NSScreen.screens.first
        guard let visible = screen?.visibleFrame else {
            panel.center()
            return
        }
        let size = host.fittingSize
        let origin = SpaceLabelOverlayLayout.origin(x: x, y: y, size: size, visible: visible)
        applyingFrame = true
        panel.setFrame(NSRect(origin: origin, size: size), display: true)
        applyingFrame = false
        host.frame = NSRect(origin: .zero, size: size)
    }

    @objc private func windowDidMove(_ notification: Notification) {
        guard !applyingFrame, !editing else { return }
        persistWork?.cancel()
        let work = DispatchWorkItem { [weak self] in
            self?.emitMove()
        }
        persistWork = work
        DispatchQueue.main.asyncAfter(deadline: .now() + 0.25, execute: work)
    }

    @objc private func windowDidResignKey(_ notification: Notification) {
        if editing {
            commitEdit()
        }
    }

    private func emitMove() {
        let screen = panel.screen ?? NSScreen.main ?? NSScreen.screens.first
        guard let visible = screen?.visibleFrame else { return }
        onMoved?(panel.frame.origin, panel.frame.size, visible)
    }

    private func showMenu() {
        guard SpaceLabelOverlayMenu.shouldPresentMenu(editing: editing, menuRefreshing: menuRefreshing) else { return }
        let menu = NSMenu()
        menu.autoenablesItems = false
        openMenu = menu
        let startRefresh = onRefreshSpace != nil && SpaceLabelOverlayMenu.shouldStartRefresh(alreadyRefreshing: menuRefreshing)
        if startRefresh {
            menuRefreshing = true
        }
        applyRows(to: menu, sessions: sessions, refreshing: menuRefreshing)
        if startRefresh, let refresh = onRefreshSpace {
            let spaceID = self.spaceID
            Task {
                let next = await refresh(spaceID)
                DispatchQueue.main.async { [weak self] in
                    guard let self else { return }
                    MainActor.assumeIsolated {
                        self.menuRefreshing = false
                        let previous = self.sessions
                        if let next {
                            self.sessions = next
                        }
                        guard let current = self.openMenu else { return }
                        if SpaceLabelOverlayMenu.shouldReplaceMenu(before: previous, after: self.sessions) {
                            self.applyRows(to: current, sessions: self.sessions, refreshing: false)
                        } else {
                            self.clearRefreshingMark(on: current)
                        }
                    }
                }
            }
        }
        let point = NSPoint(x: host.bounds.midX, y: 0)
        menu.popUp(positioning: nil, at: point, in: host)
        if openMenu === menu {
            openMenu = nil
        }
    }

    private func clearRefreshingMark(on menu: NSMenu) {
        guard let first = menu.items.first else { return }
        let mark = SpaceLabelOverlayMenu.refreshingMark
        guard first.title.hasSuffix(mark) else { return }
        first.title = String(first.title.dropLast(mark.count))
    }

    private func applyRows(to menu: NSMenu, sessions: [ITermLiveSession], refreshing: Bool) {
        menu.removeAllItems()
        for row in SpaceLabelOverlayMenu.rows(sessions: sessions, refreshing: refreshing) {
            switch row {
            case .status(let title), .empty(let title), .window(let title):
                let item = NSMenuItem(title: title, action: nil, keyEquivalent: "")
                item.isEnabled = false
                menu.addItem(item)
            case .separator:
                menu.addItem(.separator())
            case .tab(let title, let sessionID):
                let item = NSMenuItem(title: title, action: #selector(focusTab(_:)), keyEquivalent: "")
                item.target = self
                item.representedObject = sessionID
                item.isEnabled = !sessionID.isEmpty
                item.indentationLevel = 1
                menu.addItem(item)
            }
        }
    }

    @objc private func focusTab(_ sender: NSMenuItem) {
        guard let id = sender.representedObject as? String, !id.isEmpty else { return }
        onFocus?(id)
    }
}

private struct SpaceLabelPill: View {
    let text: String
    let editing: Bool
    let draft: String
    let onDraft: (String) -> Void
    let onCommit: () -> Void
    let onCancel: () -> Void
    let onBeginEdit: () -> Void
    let onDismiss: () -> Void
    let onOpenMenu: () -> Void
    @FocusState private var fieldFocused: Bool

    var body: some View {
        Group {
            if editing {
                TextField("", text: Binding(get: { draft }, set: onDraft))
                    .textFieldStyle(.plain)
                    .focused($fieldFocused)
                    .onSubmit(onCommit)
                    .onExitCommand(perform: onCancel)
                    .frame(minWidth: 80)
                    .onAppear { fieldFocused = true }
            } else {
                HStack(spacing: 6) {
                    Text(text)
                        .onTapGesture(count: 2, perform: onBeginEdit)
                    Button(action: onOpenMenu) {
                        Image(systemName: "arrowtriangle.down.fill")
                            .font(.system(size: 8, weight: .semibold))
                            .foregroundStyle(.secondary)
                            .frame(
                                width: SpaceLabelOverlayMenu.menuButtonHitSize,
                                height: SpaceLabelOverlayMenu.menuButtonHitSize
                            )
                            .contentShape(Rectangle())
                    }
                    .buttonStyle(.plain)
                    .contentShape(Rectangle())
                    .accessibilityIdentifier("space-label-overlay-menu")
                    .accessibilityLabel(SpaceLabelOverlayMenu.formatMenuButton())
                }
            }
        }
        .font(.system(size: 13, weight: .semibold))
        .padding(.horizontal, 12)
        .padding(.vertical, 6)
        .background(.ultraThinMaterial, in: Capsule())
        .overlay(
            Capsule().strokeBorder(Color.primary.opacity(0.12), lineWidth: 0.5)
        )
        .fixedSize()
        .contextMenu {
            Button(ITermSwitcherFormatter.formatEditSpaceLabel(), action: onBeginEdit)
            Button(ITermSwitcherFormatter.formatDismissSpaceLabel(), action: onDismiss)
        }
        .accessibilityIdentifier("space-label-overlay-pill")
    }
}
