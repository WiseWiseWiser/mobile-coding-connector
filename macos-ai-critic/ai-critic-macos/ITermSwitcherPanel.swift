import AppKit
import SwiftUI
import AICriticMacShared

/// Compact titled switcher panel for the local iTerm bookmark manager.
@available(macOS 15.0, *)
@MainActor
final class ITermSwitcherController {
    static let shared = ITermSwitcherController()

    private var panel: NSPanel?
    private var hosting: NSHostingView<ITermSwitcherView>?
    /// Clicks in this process but outside the panel (e.g. the main window).
    private var localClickMonitor: Any?
    /// Clicks in other apps / desktop. Mouse monitors do not need Accessibility.
    private var globalClickMonitor: Any?
    private var lastInsideClickUptime: TimeInterval = 0

    var isVisible: Bool { panel?.isVisible == true }

    func toggle() {
        if isVisible {
            hide()
        } else {
            show()
        }
    }

    func show() {
        SkillsPickerController.shared.hide()
        NSApp.setActivationPolicy(.regular)
        NSApp.activate(ignoringOtherApps: true)
        let view = ITermSwitcherView(onDismiss: { [weak self] in
            self?.hide()
        })
        if let panel {
            let host = NSHostingView(rootView: view)
            host.frame = NSRect(x: 0, y: 0, width: 720, height: 480)
            panel.contentView = host
            hosting = host
            panel.collectionBehavior = [.canJoinAllSpaces, .fullScreenAuxiliary]
            position(panel)
            panel.makeKeyAndOrderFront(nil)
            startClickOutsideMonitor()
            ITermSwitcherDebug.log("panel show reused")
            return
        }
        let panel = NSPanel(
            contentRect: NSRect(x: 0, y: 0, width: 720, height: 480),
            styleMask: [.titled, .closable, .nonactivatingPanel],
            backing: .buffered,
            defer: false
        )
        panel.title = ITermSwitcherFormatter.formatWindowTitle()
        panel.isFloatingPanel = true
        panel.level = .floating
        panel.hidesOnDeactivate = true
        panel.isOpaque = true
        panel.backgroundColor = .windowBackgroundColor
        panel.titleVisibility = .visible
        panel.titlebarAppearsTransparent = false
        panel.isMovableByWindowBackground = true
        panel.collectionBehavior = [.canJoinAllSpaces, .fullScreenAuxiliary]
        let host = NSHostingView(rootView: view)
        host.frame = NSRect(x: 0, y: 0, width: 720, height: 480)
        panel.contentView = host
        panel.isReleasedWhenClosed = false
        self.panel = panel
        self.hosting = host
        position(panel)
        panel.makeKeyAndOrderFront(nil)
        startClickOutsideMonitor()
        ITermSwitcherDebug.log("panel show created")
    }

    func hide() {
        stopClickOutsideMonitor()
        panel?.orderOut(nil)
    }

    /// Drop key so SkyLight in the daemon can change the user's Space.
    /// A key canJoinAllSpaces panel makes SetCurrentSpace a no-op / snap-back.
    func withdrawForSpaceSwitch() {
        ITermSwitcherDebug.log("panel withdraw for space switch")
        panel?.orderOut(nil)
    }

    func restoreAfterSpaceSwitch() {
        guard let panel else { return }
        ITermSwitcherDebug.log("panel restore after space switch")
        panel.collectionBehavior = [.canJoinAllSpaces, .fullScreenAuxiliary]
        position(panel)
        panel.orderFrontRegardless()
    }

    /// Spotlight-style dismiss: any mouse-down whose screen point is outside
    /// the panel. `hidesOnDeactivate` alone is not enough — LSUIElement apps
    /// stay active when the user clicks the desktop or another of our windows.
    private func startClickOutsideMonitor() {
        stopClickOutsideMonitor()
        let mask: NSEvent.EventTypeMask = [.leftMouseDown, .rightMouseDown]
        localClickMonitor = NSEvent.addLocalMonitorForEvents(matching: mask) { [weak self] event in
            guard let self else { return event }
            if event.type == .leftMouseDown {
                self.handleInsideClick(event)
            }
            self.dismissIfClickOutside()
            return event
        }
        globalClickMonitor = NSEvent.addGlobalMonitorForEvents(matching: mask) { [weak self] _ in
            self?.dismissIfClickOutside()
        }
    }

    private func stopClickOutsideMonitor() {
        if let localClickMonitor {
            NSEvent.removeMonitor(localClickMonitor)
            self.localClickMonitor = nil
        }
        if let globalClickMonitor {
            NSEvent.removeMonitor(globalClickMonitor)
            self.globalClickMonitor = nil
        }
    }

    private func handleInsideClick(_ event: NSEvent) {
        guard let panel, panel.isVisible else { return }
        guard panel.frame.contains(NSEvent.mouseLocation) else {
            lastInsideClickUptime = 0
            return
        }
        // Sidebar double-click switches Space; it must not focus a leftover session.
        if isSidebarClick(event, panel: panel) {
            lastInsideClickUptime = 0
            return
        }
        let now = ProcessInfo.processInfo.systemUptime
        let interval = NSEvent.doubleClickInterval
        let isDouble = event.clickCount >= 2 || (lastInsideClickUptime > 0 && now - lastInsideClickUptime <= interval)
        lastInsideClickUptime = now
        guard isDouble else { return }
        lastInsideClickUptime = 0
        ITermFocusHook.shared.fire()
    }

    /// Sidebar column is leading, max 200pt + splitter. Session-focus is detail-only.
    private func isSidebarClick(_ event: NSEvent, panel: NSPanel) -> Bool {
        let loc = panel.convertPoint(fromScreen: NSEvent.mouseLocation)
        return loc.x < 208
    }

    private func dismissIfClickOutside() {
        guard let panel, panel.isVisible else { return }
        if !panel.frame.contains(NSEvent.mouseLocation) {
            hide()
        }
    }

    private func position(_ panel: NSPanel) {
        if let screen = NSScreen.main {
            let frame = screen.visibleFrame
            let size = panel.frame.size
            let x = frame.midX - size.width / 2
            let y = frame.midY - size.height / 2 + 40
            panel.setFrameOrigin(NSPoint(x: x, y: y))
        } else {
            panel.center()
        }
    }
}

@MainActor
final class ITermFocusHook {
    static let shared = ITermFocusHook()
    var sessionID: String?
    var blocked = false
    var run: ((String) async -> Void)?

    func fire() {
        log("fire blocked=\(blocked) id=\(sessionID ?? "nil") run=\(run != nil)")
        guard !blocked, let sessionID, let run else { return }
        Task { await run(sessionID) }
    }

    func log(_ message: String) {
        ITermSwitcherDebug.log(message)
    }
}

@available(macOS 15.0, *)
struct ITermSwitcherView: View {
    let onDismiss: () -> Void

    @State private var query = ""
    @State private var inventory = ITermInventory()
    @State private var sidebarID: String? = ITermSwitcherFormatter.sidebarAll
    @State private var selectedID: String?
    @State private var editingID: String?
    @State private var draftNote = ""
    @State private var editingSpaceLabel = false
    @State private var draftSpaceLabel = ""
    @State private var errorText: String?
    @State private var showAccessibilitySettings = false
    @State private var loading = true
    @State private var bookmarkOverrides: [String: Bool] = [:]
    @State private var didApplyInitialDesktop = false
    @FocusState private var noteFocused: Bool
    @ObservedObject private var overlayChrome = SpaceLabelOverlayController.shared

    var body: some View {
        NavigationSplitView {
            sidebar
        } detail: {
            detail
        }
        .navigationTitle(ITermSwitcherFormatter.formatWindowTitle())
        .searchable(text: $query, prompt: "Search terminals, notes, cwd…")
        .toolbar {
            if inventory.refreshing {
                ToolbarItem(placement: .automatic) {
                    ProgressView()
                        .controlSize(.small)
                        .accessibilityIdentifier("iterm-switcher-refreshing")
                }
            }
        }
        .frame(width: 720, height: 480)
        .accessibilityIdentifier("iterm-switcher")
        .onAppear {
            bindFocusHook()
            Task { await reload() }
        }
        .onReceive(NSWorkspace.shared.notificationCenter.publisher(for: NSWorkspace.activeSpaceDidChangeNotification)) { _ in
            Task { await refreshCurrentSpace() }
        }
        .onChange(of: selectedID) { _, _ in
            bindFocusHook()
        }
        .onChange(of: sidebarID) { _, _ in
            editingSpaceLabel = false
            draftSpaceLabel = ""
            bindFocusHook()
        }
        .onKeyPress(.delete) {
            if selectingSpaceLabel && !editingSpaceLabel {
                Task { await clearSpaceLabel() }
                return .handled
            }
            return .ignored
        }
        .onExitCommand { handleEscape() }
        .onSubmit(of: .search) { activateSelection() }
        .onKeyPress(.return) {
            activateSelection()
            return .handled
        }
        .onKeyPress(.upArrow) {
            moveSelection(-1)
            return .handled
        }
        .onKeyPress(.downArrow) {
            moveSelection(1)
            return .handled
        }
        .onKeyPress(.escape) {
            handleEscape()
            return .handled
        }
        .onKeyPress(keys: [.init("n")]) { press in
            if press.modifiers.contains(.command) {
                beginEditSelected()
                return .handled
            }
            return .ignored
        }
        .onKeyPress(keys: [.init("d")]) { press in
            if press.modifiers.contains(.command) {
                toggleBookmarkSelected()
                return .handled
            }
            return .ignored
        }
    }

    private var sidebar: some View {
        ScrollViewReader { proxy in
            List(selection: $sidebarID) {
                Label(ITermSwitcherFormatter.formatSidebarTitle(id: ITermSwitcherFormatter.sidebarAll), systemImage: "square.grid.2x2")
                    .tag(ITermSwitcherFormatter.sidebarAll)
                    .id(ITermSwitcherFormatter.sidebarAll)
                Label(ITermSwitcherFormatter.formatSidebarTitle(id: ITermSwitcherFormatter.sidebarBookmarks), systemImage: "star")
                    .badge(bookmarkCount)
                    .tag(ITermSwitcherFormatter.sidebarBookmarks)
                    .id(ITermSwitcherFormatter.sidebarBookmarks)
                ForEach(inventory.desktops) { group in
                    let deskID = ITermSwitcherFormatter.sidebarDesktopID(spaceIndex: group.spaceIndex)
                    Label {
                        Text(ITermSwitcherFormatter.formatSidebarDesktopTitle(spaceIndex: group.spaceIndex, label: group.label))
                    } icon: {
                        Image(systemName: ITermSwitcherFormatter.formatDesktopSidebarSymbol(current: group.current))
                            .symbolRenderingMode(.monochrome)
                            .foregroundStyle(group.current ? Color.accentColor : Color.secondary)
                    }
                    .badge(group.sessions.count)
                    .tag(deskID)
                    .id(deskID)
                    .contentShape(Rectangle())
                    .simultaneousGesture(TapGesture(count: 2).onEnded {
                        switchToDesktop(group)
                    })
                }
                Label(ITermSwitcherFormatter.formatSidebarTitle(id: ITermSwitcherFormatter.sidebarSaved), systemImage: "tray")
                    .badge(inventory.savedNotes.count)
                    .tag(ITermSwitcherFormatter.sidebarSaved)
                    .id(ITermSwitcherFormatter.sidebarSaved)
            }
            .listStyle(.sidebar)
            .navigationSplitViewColumnWidth(min: 140, ideal: 160, max: 200)
            .accessibilityIdentifier("iterm-switcher-sidebar")
            .onChange(of: sidebarID) { _, id in
                scrollSidebar(proxy, to: id)
            }
            .onChange(of: didApplyInitialDesktop) { _, applied in
                guard applied else { return }
                scrollSidebar(proxy, to: sidebarID)
            }
        }
    }

    private func scrollSidebar(_ proxy: ScrollViewProxy, to id: String?) {
        guard let id, !id.isEmpty else { return }
        DispatchQueue.main.async {
            withAnimation(.easeInOut(duration: 0.2)) {
                proxy.scrollTo(id, anchor: .center)
            }
        }
    }

    @ViewBuilder
    private var detail: some View {
        VStack(alignment: .leading, spacing: 0) {
            if let errorText {
                HStack(alignment: .top, spacing: 8) {
                    Image(systemName: "exclamationmark.triangle.fill")
                        .foregroundStyle(.yellow)
                    Text(errorText)
                        .font(.caption)
                        .fixedSize(horizontal: false, vertical: true)
                    Spacer()
                    if showAccessibilitySettings {
                        Button("Open Settings") {
                            if let url = URL(string: "x-apple.systempreferences:com.apple.preference.security?Privacy_Accessibility") {
                                NSWorkspace.shared.open(url)
                            }
                        }
                        .controlSize(.small)
                    }
                }
                .padding(10)
                .background(Color.yellow.opacity(0.12))
            }

            if loading && liveSessions.isEmpty && inventory.savedNotes.isEmpty {
                ContentUnavailableView {
                    ProgressView()
                } description: {
                    Text("Loading…")
                }
                .frame(maxWidth: .infinity, maxHeight: .infinity)
            } else if showingSaved {
                savedList
            } else if showingDesktop {
                desktopList
            } else if !inventory.itermRunning && liveSessions.isEmpty {
                ContentUnavailableView(
                    ITermSwitcherFormatter.formatEmptyITerm(),
                    systemImage: "apple.terminal",
                    description: Text("Open iTerm, then press \(ITermSwitcherFormatter.formatDefaultHotKey()) again")
                )
            } else if filteredSessions.isEmpty {
                ContentUnavailableView(
                    emptyDetailTitle,
                    systemImage: resolvedSidebar == ITermSwitcherFormatter.sidebarBookmarks ? "star" : "magnifyingglass",
                    description: Text(emptyDetailSubtitle)
                )
            } else {
                sessionList
            }
        }
        .accessibilityIdentifier("iterm-switcher-detail")
    }

    private var sessionList: some View {
        List(filteredSessions, selection: $selectedID) { sess in
            sessionRow(sess)
                .tag(sess.sessionID)
                .id(sess.sessionID)
        }
    }

    private var desktopList: some View {
        List(selection: $selectedID) {
            Section {
                spaceLabelRow
                    .tag(ITermSwitcherFormatter.spaceLabelRowID)
                    .id(ITermSwitcherFormatter.spaceLabelRowID)
            }
            Section {
                if filteredSessions.isEmpty {
                    ContentUnavailableView(
                        emptyDetailTitle,
                        systemImage: "magnifyingglass",
                        description: Text(emptyDetailSubtitle)
                    )
                    .selectionDisabled()
                } else {
                    ForEach(filteredSessions) { sess in
                        sessionRow(sess)
                            .tag(sess.sessionID)
                            .id(sess.sessionID)
                    }
                }
            }
        }
    }

    @ViewBuilder
    private var spaceLabelRow: some View {
        let current = currentDesktop?.label ?? ""
        if editingSpaceLabel {
            VStack(alignment: .leading, spacing: 6) {
                Text(ITermSwitcherFormatter.formatSpaceLabelRow(current))
                    .fontWeight(.medium)
                    .foregroundStyle(.secondary)
                TextField(ITermSwitcherFormatter.formatSpaceLabelRow(""), text: $draftSpaceLabel)
                    .textFieldStyle(.roundedBorder)
                    .focused($noteFocused)
                    .onSubmit { Task { await saveSpaceLabel() } }
                    .accessibilityIdentifier("iterm-switcher-space-label-field")
            }
        } else {
            HStack(spacing: 8) {
                Image(systemName: "tag")
                    .foregroundStyle(.secondary)
                    .frame(width: 22, height: 22)
                Text(ITermSwitcherFormatter.formatSpaceLabelRow(current))
                    .fontWeight(.medium)
                Spacer()
                if !current.trimmingCharacters(in: .whitespacesAndNewlines).isEmpty {
                    Button(ITermSwitcherFormatter.formatChangeSpaceLabel()) {
                        beginEditSpaceLabel()
                    }
                    .controlSize(.small)
                    Button(ITermSwitcherFormatter.formatClearSpaceLabel()) {
                        Task { await clearSpaceLabel() }
                    }
                    .controlSize(.small)
                    if spaceLabelOverlayHidden {
                        Button(ITermSwitcherFormatter.formatShowSpaceLabel()) {
                            showSpaceLabelOverlay()
                        }
                        .controlSize(.small)
                    }
                }
            }
            .contentShape(Rectangle())
            .onTapGesture {
                if current.trimmingCharacters(in: .whitespacesAndNewlines).isEmpty {
                    beginEditSpaceLabel()
                }
            }
            .accessibilityIdentifier("iterm-switcher-space-label")
        }
    }

    private var savedList: some View {
        Group {
            if filteredOrphans.isEmpty {
                ContentUnavailableView(
                    "No saved notes",
                    systemImage: "tray",
                    description: Text("Gone bookmarked tabs and notes appear here")
                )
            } else {
                List(filteredOrphans, selection: $selectedID) { orphan in
                    VStack(alignment: .leading, spacing: 2) {
                        HStack {
                            if orphan.bookmarked {
                                Image(systemName: "star.fill")
                                    .foregroundStyle(.yellow)
                            }
                            Text(ITermSwitcherFormatter.formatOrphanPrimary(
                                note: orphan.note,
                                sessionName: orphan.sessionName,
                                cwd: orphan.cwd,
                                sessionID: orphan.sessionID
                            ))
                            .fontWeight(.medium)
                        }
                        Text(orphanSubtitle(orphan))
                            .font(.caption)
                            .foregroundStyle(.secondary)
                        Button(orphan.bookmarked && orphan.note.isEmpty ? "Remove bookmark" : "Delete note") {
                            Task { await deleteOrphan(orphan) }
                        }
                        .controlSize(.small)
                    }
                    .tag(orphan.sessionID)
                    .id(orphan.sessionID)
                }
            }
        }
    }

    private func sessionRow(_ sess: ITermLiveSession) -> some View {
        let starred = sessionBookmarked(sess)
        return HStack(alignment: .top, spacing: 8) {
            Button {
                Task { await toggleBookmark(sess) }
            } label: {
                Image(systemName: starred ? "star.fill" : "star")
                    .foregroundStyle(starred ? Color.yellow : Color.secondary)
                    .frame(width: 22, height: 22)
                    .contentShape(Rectangle())
            }
            .buttonStyle(.borderless)
            .help(starred ? "Remove Bookmark" : "Bookmark")
            .accessibilityIdentifier("iterm-switcher-star-\(sess.sessionID)")

            VStack(alignment: .leading, spacing: 4) {
                if editingID == sess.sessionID {
                    Text(ITermSwitcherFormatter.formatSessionPrimary(name: sess.sessionName, cwd: sess.cwd, sessionID: sess.sessionID))
                        .fontWeight(.medium)
                    TextField("Note", text: $draftNote)
                        .textFieldStyle(.roundedBorder)
                        .focused($noteFocused)
                        .onSubmit { Task { await saveNote() } }
                } else {
                    HStack {
                        Text(ITermSwitcherFormatter.formatSessionPrimary(name: sess.sessionName, cwd: sess.cwd, sessionID: sess.sessionID))
                            .fontWeight(.medium)
                        Spacer()
                        Text(ITermSwitcherFormatter.formatSessionNote(sess.note))
                            .foregroundStyle(.secondary)
                    }
                }
                Text(ITermSwitcherFormatter.formatSessionSubtitle(cwd: sess.cwd, tabIndex: sess.tabIndex, idle: sess.idle))
                    .font(.caption)
                    .foregroundStyle(.secondary)
            }
            .frame(maxWidth: .infinity, alignment: .leading)
            .contentShape(Rectangle())
        }
        .contextMenu {
            Button(starred ? "Remove Bookmark" : "Bookmark") {
                Task { await toggleBookmark(sess) }
            }
            Button("Edit Note") {
                selectedID = sess.sessionID
                beginEditSelected()
            }
        }
    }

    private var resolvedSidebar: String {
        sidebarID ?? ITermSwitcherFormatter.sidebarAll
    }

    private var showingSaved: Bool {
        resolvedSidebar == ITermSwitcherFormatter.sidebarSaved
    }

    private var showingDesktop: Bool {
        ITermSwitcherFormatter.parseSidebarDesktop(resolvedSidebar) != nil
    }

    private var currentDesktop: ITermDesktopGroup? {
        guard let idx = ITermSwitcherFormatter.parseSidebarDesktop(resolvedSidebar) else { return nil }
        return inventory.desktops.first { $0.spaceIndex == idx }
    }

    private func desktopLabel(for spaceIndex: Int) -> String {
        inventory.desktops.first { $0.spaceIndex == spaceIndex }?.label ?? ""
    }

    private var liveSessions: [ITermLiveSession] {
        inventory.desktops.flatMap(\.sessions)
    }

    private func sessionBookmarked(_ sess: ITermLiveSession) -> Bool {
        ITermSwitcherFormatter.resolvedBookmarked(
            sessionID: sess.sessionID,
            inventoryValue: sess.bookmarked,
            overrides: bookmarkOverrides
        )
    }

    private var bookmarkCount: Int {
        liveSessions.filter { sessionBookmarked($0) }.count
    }

    private var filteredSessions: [ITermLiveSession] {
        liveSessions.filter { sess in
            ITermSwitcherFormatter.matchesSidebar(
                id: resolvedSidebar,
                spaceIndex: sess.spaceIndex,
                bookmarked: sessionBookmarked(sess)
            ) && ITermSwitcherFormatter.sessionMatches(
                name: sess.sessionName,
                note: sess.note,
                cwd: sess.cwd,
                windowName: sess.windowName,
                tabName: sess.tabName,
                sessionID: sess.sessionID,
                spaceIndex: sess.spaceIndex,
                spaceLabel: desktopLabel(for: sess.spaceIndex),
                query: query
            )
        }
    }

    private var filteredOrphans: [ITermOrphanNote] {
        inventory.savedNotes.filter { ITermSwitcherFormatter.orphanMatches($0, query: query) }
    }

    private var emptyDetailTitle: String {
        if resolvedSidebar == ITermSwitcherFormatter.sidebarBookmarks {
            return "No Bookmarks"
        }
        if !query.trimmingCharacters(in: .whitespacesAndNewlines).isEmpty {
            return "No Results"
        }
        return "No Terminals"
    }

    private var emptyDetailSubtitle: String {
        if resolvedSidebar == ITermSwitcherFormatter.sidebarBookmarks {
            return "Press ⌘D to bookmark a tab"
        }
        return "↑↓ select   ⏎ switch   ⌘D bookmark   ⌘N note"
    }

    private func orphanSubtitle(_ orphan: ITermOrphanNote) -> String {
        var parts = ["was:"]
        if !orphan.cwd.isEmpty { parts.append(orphan.cwd) }
        parts.append(ITermSwitcherFormatter.formatDesktopHeader(spaceIndex: orphan.spaceIndex))
        parts.append("gone")
        return parts.joined(separator: "  ·  ")
    }

    private var visibleIDs: [String] {
        if showingSaved {
            return filteredOrphans.map(\.sessionID)
        }
        var ids: [String] = []
        if showingDesktop {
            ids.append(ITermSwitcherFormatter.spaceLabelRowID)
        }
        ids.append(contentsOf: filteredSessions.map(\.sessionID))
        return ids
    }

    private var selectingSpaceLabel: Bool {
        selectedID == ITermSwitcherFormatter.spaceLabelRowID
    }

    private func handleEscape() {
        if editingSpaceLabel {
            editingSpaceLabel = false
            draftSpaceLabel = ""
            return
        }
        if editingID != nil {
            editingID = nil
            draftNote = ""
            return
        }
        if !query.isEmpty {
            query = ""
            return
        }
        onDismiss()
    }

    private func moveSelection(_ delta: Int) {
        let items = visibleIDs
        guard !items.isEmpty else { return }
        let current = items.firstIndex(where: { $0 == selectedID }) ?? (delta > 0 ? -1 : 0)
        var next = current + delta
        if next < 0 { next = 0 }
        if next >= items.count { next = items.count - 1 }
        selectedID = items[next]
    }

    private func beginEditSelected() {
        if selectingSpaceLabel || (selectedID == nil && showingDesktop && filteredSessions.isEmpty) {
            beginEditSpaceLabel()
            return
        }
        guard let id = selectedID, let sess = filteredSessions.first(where: { $0.sessionID == id }) else { return }
        editingID = id
        draftNote = sess.note
        noteFocused = true
    }

    private func beginEditSpaceLabel() {
        selectedID = ITermSwitcherFormatter.spaceLabelRowID
        editingSpaceLabel = true
        draftSpaceLabel = currentDesktop?.label ?? ""
        noteFocused = true
    }

    private func toggleBookmarkSelected() {
        guard let id = selectedID, let sess = filteredSessions.first(where: { $0.sessionID == id }) else { return }
        Task { await toggleBookmark(sess) }
    }

    private func bindFocusHook() {
        ITermFocusHook.shared.sessionID = selectedID ?? filteredSessions.first?.sessionID
        ITermFocusHook.shared.blocked = showingSaved || editingID != nil || editingSpaceLabel || selectingSpaceLabel
        ITermFocusHook.shared.run = { id in
            await focus(id)
        }
    }

    private func activateSelection() {
        if editingSpaceLabel {
            Task { await saveSpaceLabel() }
            return
        }
        if selectingSpaceLabel {
            beginEditSpaceLabel()
            return
        }
        if editingID != nil {
            Task { await saveNote() }
            return
        }
        if showingSaved {
            return
        }
        guard let id = selectedID else {
            if let first = filteredSessions.first {
                selectedID = first.sessionID
                Task { await focus(first.sessionID) }
            }
            return
        }
        Task { await focus(id) }
    }

    private func reload() async {
        loading = liveSessions.isEmpty && inventory.savedNotes.isEmpty
        errorText = nil
        showAccessibilitySettings = false
        do {
            for try await inv in ServerClient.shared.itermInventoryStream() {
                inventory = inv
                reconcileBookmarkOverrides(from: inv)
                SpaceLabelOverlayController.shared.apply(desktops: inv.desktops)
                loading = false
                applyInitialDesktopIfNeeded()
                normalizeSidebar()
                if selectedID == nil {
                    selectedID = initialSelectedSessionID()
                }
                bindFocusHook()
            }
        } catch {
            errorText = error.localizedDescription
        }
        loading = false
    }

    private func refreshCurrentSpace() async {
        do {
            let inv = try await ServerClient.shared.itermInventory()
            inventory = inv
            SpaceLabelOverlayController.shared.apply(desktops: inv.desktops)
        } catch {
            // Space change refresh is best-effort.
        }
    }

    private func applyInitialDesktopIfNeeded() {
        guard !didApplyInitialDesktop else { return }
        guard let current = inventory.desktops.first(where: { $0.current }) else { return }
        sidebarID = ITermSwitcherFormatter.initialSidebarID(currentSpaceIndex: current.spaceIndex)
        selectedID = current.sessions.first?.sessionID
        didApplyInitialDesktop = true
    }

    private func initialSelectedSessionID() -> String? {
        if let current = inventory.desktops.first(where: { $0.current }),
           let id = current.sessions.first?.sessionID {
            return id
        }
        return inventory.desktops.flatMap(\.sessions).first?.sessionID
    }

    private func normalizeSidebar() {
        if let idx = ITermSwitcherFormatter.parseSidebarDesktop(resolvedSidebar),
           !inventory.desktops.contains(where: { $0.spaceIndex == idx }) {
            sidebarID = ITermSwitcherFormatter.sidebarAll
        }
    }

    private func switchToDesktop(_ group: ITermDesktopGroup) {
        let deskID = ITermSwitcherFormatter.sidebarDesktopID(spaceIndex: group.spaceIndex)
        ITermSwitcherDebug.log(
            "switchToDesktop deskID=\(deskID) spaceIndex=\(group.spaceIndex) spaceID=\(group.spaceID) current=\(group.current)"
        )
        guard ITermSwitcherFormatter.shouldSwitchSpace(sidebarID: deskID) else { return }
        if group.current { return }
        guard group.spaceID != 0 else {
            errorText = ITermSwitcherFormatter.formatSwitchSpaceMissingID()
            return
        }
        let spaceID = group.spaceID
        let fromIndex = inventory.desktops.first(where: { $0.current })?.spaceIndex
        let toIndex = group.spaceIndex
        let firstSessionID = group.sessions.first?.sessionID
        ITermSwitcherController.shared.withdrawForSpaceSwitch()
        SpaceSwitch.go(
            spaceID: spaceID,
            fromIndex: fromIndex,
            toIndex: toIndex,
            firstSessionID: firstSessionID,
            focusSession: { id in await focus(id) }
        ) {
            if CGSSpaceMove.activeSpaceID() == spaceID {
                ITermSwitcherController.shared.restoreAfterSpaceSwitch()
            } else {
                ITermSwitcherDebug.log("SpaceSwitch leave popup hidden; space did not change")
                ITermSwitcherController.shared.restoreAfterSpaceSwitch()
            }
        }
    }

    private func focus(_ sessionID: String) async {
        ITermFocusHook.shared.log("focus start id=\(sessionID)")
        do {
            try await ServerClient.shared.focusITermSession(sessionID: sessionID)
            ITermFocusHook.shared.log("focus ok")
            onDismiss()
        } catch {
            let msg = error.localizedDescription
            ITermFocusHook.shared.log("focus err \(msg)")
            errorText = msg
            showAccessibilitySettings = msg.localizedCaseInsensitiveContains("Accessibility")
        }
    }

    private func reconcileBookmarkOverrides(from inv: ITermInventory) {
        var live: [String: Bool] = [:]
        for sess in inv.desktops.flatMap(\.sessions) {
            live[sess.sessionID] = sess.bookmarked
        }
        bookmarkOverrides = ITermSwitcherFormatter.reconcileBookmarkOverrides(
            overrides: bookmarkOverrides,
            live: live
        )
    }

    private func toggleBookmark(_ sess: ITermLiveSession) async {
        let next = !sessionBookmarked(sess)
        bookmarkOverrides[sess.sessionID] = next
        selectedID = sess.sessionID
        do {
            try await ServerClient.shared.setITermNote(sessionID: sess.sessionID, bookmarked: next)
        } catch {
            bookmarkOverrides.removeValue(forKey: sess.sessionID)
            errorText = error.localizedDescription
        }
    }

    private func saveNote() async {
        guard let id = editingID else { return }
        do {
            try await ServerClient.shared.setITermNote(sessionID: id, note: draftNote)
            editingID = nil
            draftNote = ""
            await reload()
            selectedID = id
        } catch {
            errorText = error.localizedDescription
        }
    }

    private func deleteOrphan(_ orphan: ITermOrphanNote) async {
        do {
            try await ServerClient.shared.setITermNote(sessionID: orphan.sessionID, note: "", bookmarked: false)
            await reload()
        } catch {
            errorText = error.localizedDescription
        }
    }

    private func saveSpaceLabel() async {
        guard let desk = currentDesktop else {
            editingSpaceLabel = false
            return
        }
        if desk.spaceID == 0 && desk.spaceUUID.isEmpty {
            errorText = "Can't resolve this Space"
            return
        }
        let next = draftSpaceLabel
        do {
            try await ServerClient.shared.setITermSpaceLabel(
                spaceID: desk.spaceID,
                uuid: desk.spaceUUID,
                label: next
            )
            if !next.trimmingCharacters(in: .whitespacesAndNewlines).isEmpty {
                SpaceLabelOverlayController.shared.reveal(spaceID: desk.spaceID, uuid: desk.spaceUUID)
            }
            editingSpaceLabel = false
            draftSpaceLabel = ""
            await reload()
            selectedID = ITermSwitcherFormatter.spaceLabelRowID
        } catch {
            errorText = error.localizedDescription
        }
    }

    private var spaceLabelOverlayHidden: Bool {
        guard let desk = currentDesktop else { return false }
        _ = overlayChrome.hiddenIDs
        return SpaceLabelOverlayController.shared.isHidden(spaceID: desk.spaceID, uuid: desk.spaceUUID)
    }

    private func showSpaceLabelOverlay() {
        guard let desk = currentDesktop else { return }
        SpaceLabelOverlayController.shared.reveal(spaceID: desk.spaceID, uuid: desk.spaceUUID)
        SpaceLabelOverlayController.shared.apply(desktops: inventory.desktops)
    }

    private func clearSpaceLabel() async {
        guard let desk = currentDesktop else { return }
        if desk.spaceID == 0 && desk.spaceUUID.isEmpty {
            errorText = "Can't resolve this Space"
            return
        }
        do {
            try await ServerClient.shared.setITermSpaceLabel(
                spaceID: desk.spaceID,
                uuid: desk.spaceUUID,
                label: ""
            )
            editingSpaceLabel = false
            draftSpaceLabel = ""
            await reload()
            selectedID = ITermSwitcherFormatter.spaceLabelRowID
        } catch {
            errorText = error.localizedDescription
        }
    }
}
