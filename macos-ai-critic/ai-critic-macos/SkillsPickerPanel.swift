import AppKit
import SwiftUI
import AICriticMacShared

@available(macOS 15.0, *)
@MainActor
final class SkillsPickerController {
    static let shared = SkillsPickerController()

    private var panel: NSPanel?
    private var hosting: NSHostingView<SkillsPickerView>?
    private var localClickMonitor: Any?
    private var globalClickMonitor: Any?

    var isVisible: Bool { panel?.isVisible == true }

    func toggle() {
        if isVisible {
            hide()
        } else {
            show()
        }
    }

    func show() {
        ITermSwitcherController.shared.hide()
        NSApp.setActivationPolicy(.regular)
        NSApp.activate(ignoringOtherApps: true)
        let view = SkillsPickerView(onDismiss: { [weak self] in
            self?.hide()
        })
        let width: CGFloat = 640
        let height: CGFloat = 420
        if let panel {
            let host = NSHostingView(rootView: view)
            host.frame = NSRect(x: 0, y: 0, width: width, height: height)
            panel.contentView = host
            hosting = host
            panel.collectionBehavior = [.canJoinAllSpaces, .fullScreenAuxiliary]
            position(panel)
            panel.makeKeyAndOrderFront(nil)
            startClickOutsideMonitor()
            return
        }
        let panel = NSPanel(
            contentRect: NSRect(x: 0, y: 0, width: width, height: height),
            styleMask: [.titled, .closable, .nonactivatingPanel],
            backing: .buffered,
            defer: false
        )
        panel.title = SkillsPickerFormatter.formatWindowTitle()
        panel.isFloatingPanel = true
        panel.level = .floating
        panel.hidesOnDeactivate = true
        panel.isOpaque = true
        panel.backgroundColor = .windowBackgroundColor
        panel.titleVisibility = .visible
        panel.isMovableByWindowBackground = true
        panel.collectionBehavior = [.canJoinAllSpaces, .fullScreenAuxiliary]
        let host = NSHostingView(rootView: view)
        host.frame = NSRect(x: 0, y: 0, width: width, height: height)
        panel.contentView = host
        panel.isReleasedWhenClosed = false
        self.panel = panel
        self.hosting = host
        position(panel)
        panel.makeKeyAndOrderFront(nil)
        startClickOutsideMonitor()
    }

    func hide() {
        stopClickOutsideMonitor()
        panel?.orderOut(nil)
    }

    private func startClickOutsideMonitor() {
        stopClickOutsideMonitor()
        let mask: NSEvent.EventTypeMask = [.leftMouseDown, .rightMouseDown]
        localClickMonitor = NSEvent.addLocalMonitorForEvents(matching: mask) { [weak self] event in
            self?.dismissIfClickOutside()
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

@available(macOS 15.0, *)
struct SkillsPickerView: View {
    let onDismiss: () -> Void

    @AppStorage(SkillsPickerFormatter.sidebarDefaultsKey) private var storedSidebar = SkillsPickerFormatter.sidebarAll
    @AppStorage(SkillsPickerFormatter.sidebarWidthDefaultsKey) private var storedSidebarWidth = SkillsPickerFormatter.sidebarIconWidth
    @State private var sidebarID: String? = nil
    @State private var sidebarWidth: CGFloat = CGFloat(SkillsPickerFormatter.sidebarIconWidth)
    @State private var sidebarDragStartWidth: CGFloat?
    @State private var query = ""
    @State private var items: [InsertPickerItem] = []
    @State private var selectedID: String?
    @State private var loading = true
    @State private var errorText: String?
    @State private var searchTask: Task<Void, Never>?
    @FocusState private var searchFocused: Bool

    private var resolvedSidebar: String {
        SkillsPickerFormatter.normalizeSidebarID(sidebarID ?? storedSidebar)
    }

    private var showSidebarTitles: Bool {
        SkillsPickerFormatter.shouldShowSidebarTitles(width: Double(sidebarWidth))
    }

    var body: some View {
        VStack(spacing: 0) {
            searchRow
            Divider()
            HStack(spacing: 0) {
                sidebar
                    .frame(width: sidebarWidth)
                    .frame(maxHeight: .infinity)
                sidebarResizeHandle
                contentPane
                    .frame(minWidth: 280, maxWidth: .infinity, maxHeight: .infinity)
            }
        }
        .frame(width: 640, height: 420)
        .defaultFocus($searchFocused, true)
        .accessibilityIdentifier("skills-picker")
        .onAppear {
            if sidebarID == nil {
                sidebarID = SkillsPickerFormatter.normalizeSidebarID(storedSidebar)
            }
            sidebarWidth = CGFloat(SkillsPickerFormatter.clampSidebarWidth(storedSidebarWidth))
            focusSearch()
        }
        .onChange(of: query) { _, q in
            scheduleReload(q)
        }
        .onChange(of: sidebarID) { _, id in
            let normalized = SkillsPickerFormatter.normalizeSidebarID(id)
            storedSidebar = normalized
            scheduleReload(query)
        }
        .onChange(of: items.map(\.id)) { _, ids in
            if selectedID == nil || !(ids.contains(selectedID ?? "")) {
                selectedID = ids.first
            }
        }
        .onExitCommand { handleEscape() }
        .onKeyPress(.upArrow) {
            moveSelection(-1)
            return .handled
        }
        .onKeyPress(.downArrow) {
            moveSelection(1)
            return .handled
        }
        .onKeyPress(.leftArrow) {
            moveSidebar(-1)
            return .handled
        }
        .onKeyPress(.rightArrow) {
            moveSidebar(1)
            return .handled
        }
        .onKeyPress(.escape) {
            handleEscape()
            return .handled
        }
    }

    private var sidebar: some View {
        List(selection: $sidebarID) {
            sidebarRow(id: SkillsPickerFormatter.sidebarAll)
            sidebarRow(id: SkillsPickerFormatter.sidebarSkills)
            sidebarRow(id: SkillsPickerFormatter.sidebarTemplates)
            sidebarRow(id: SkillsPickerFormatter.sidebarFiles)
        }
        .listStyle(.sidebar)
        .scrollContentBackground(.hidden)
        .accessibilityIdentifier("insert-picker-sidebar")
    }

    private var sidebarResizeHandle: some View {
        ZStack {
            Rectangle()
                .fill(Color(nsColor: .separatorColor))
                .frame(width: 1)
            Color.clear
                .frame(width: 6)
                .contentShape(Rectangle())
        }
        .frame(width: 6)
        .frame(maxHeight: .infinity)
        .onHover { hovering in
            if hovering {
                NSCursor.resizeLeftRight.push()
            } else {
                NSCursor.pop()
            }
        }
        .gesture(
            DragGesture(minimumDistance: 1)
                .onChanged { value in
                    applySidebarDrag(translation: value.translation.width)
                }
                .onEnded { value in
                    applySidebarDrag(translation: value.translation.width)
                    sidebarDragStartWidth = nil
                }
        )
        .accessibilityIdentifier("insert-picker-sidebar-resize")
        .accessibilityLabel("Resize sidebar")
    }

    private func applySidebarDrag(translation: CGFloat) {
        if sidebarDragStartWidth == nil {
            sidebarDragStartWidth = sidebarWidth
        }
        guard let start = sidebarDragStartWidth else { return }
        let next = CGFloat(SkillsPickerFormatter.clampSidebarWidth(Double(start + translation)))
        // Width tracks the drag immediately; title opacity animates via showSidebarTitles.
        if abs(next - sidebarWidth) > 0.25 {
            sidebarWidth = next
        }
        let stored = SkillsPickerFormatter.clampSidebarWidth(Double(next))
        if abs(stored - storedSidebarWidth) > 0.5 {
            storedSidebarWidth = stored
        }
    }

    private func sidebarRow(id: String) -> some View {
        let title = SkillsPickerFormatter.formatSidebarTitle(id: id)
        let symbol = SkillsPickerFormatter.formatSidebarSymbol(id: id)
        return HStack(spacing: 8) {
            Image(systemName: symbol)
                .frame(width: 20, alignment: .center)
            Text(title)
                .lineLimit(1)
                .opacity(showSidebarTitles ? 1 : 0)
                .frame(maxWidth: showSidebarTitles ? .infinity : 0, alignment: .leading)
                .clipped()
        }
        .frame(maxWidth: .infinity, alignment: .leading)
        .contentShape(Rectangle())
        .tag(id)
        .id(id)
        .accessibilityLabel(title)
        .animation(
            .easeInOut(duration: SkillsPickerFormatter.sidebarTitleAnimationSeconds),
            value: showSidebarTitles
        )
    }

    @ViewBuilder
    private var contentPane: some View {
        if let errorText {
            Text(errorText)
                .foregroundStyle(.red)
                .font(.caption)
                .padding(8)
                .frame(maxWidth: .infinity, maxHeight: .infinity, alignment: .topLeading)
        } else if loading && items.isEmpty {
            ProgressView()
                .controlSize(.small)
                .frame(maxWidth: .infinity, maxHeight: .infinity)
        } else if items.isEmpty {
            VStack(spacing: 6) {
                Text(emptyTitle)
                    .font(.headline)
                if query.trimmingCharacters(in: .whitespacesAndNewlines).isEmpty {
                    Text(SkillsPickerFormatter.formatEmptyHint(sidebarID: resolvedSidebar))
                        .font(.caption)
                        .foregroundStyle(.secondary)
                }
            }
            .frame(maxWidth: .infinity, maxHeight: .infinity)
        } else {
            ScrollViewReader { proxy in
                List(items, selection: $selectedID) { item in
                    HStack(alignment: .top, spacing: 8) {
                        if resolvedSidebar == SkillsPickerFormatter.sidebarAll {
                            Text(SkillsPickerFormatter.formatKindBadge(item.kind))
                                .font(.caption2.weight(.semibold))
                                .foregroundStyle(.secondary)
                                .frame(width: 64, alignment: .leading)
                        }
                        VStack(alignment: .leading, spacing: 2) {
                            spanText(
                                item.titleSpans,
                                fallback: item.title,
                                caption: false
                            )
                            .lineLimit(1)
                            spanText(
                                item.pathSpans.isEmpty && item.kind == .template
                                    ? []
                                    : item.pathSpans,
                                fallback: item.subtitle,
                                caption: true
                            )
                            .font(.caption)
                            .lineLimit(1)
                        }
                        Spacer()
                        let count = SkillsPickerFormatter.formatUseCount(item.useCount)
                        if !count.isEmpty {
                            Text(count)
                                .font(.caption.monospacedDigit())
                                .foregroundStyle(.secondary)
                        }
                    }
                    .tag(item.id)
                    .id(item.id)
                    .contentShape(Rectangle())
                    .onTapGesture { pick(item) }
                }
                .listStyle(.sidebar)
                .scrollContentBackground(.hidden)
                .onChange(of: selectedID) { _, id in
                    scrollList(proxy, to: id)
                }
            }
        }
    }

    private func scrollList(_ proxy: ScrollViewProxy, to id: String?) {
        guard let id, !id.isEmpty else { return }
        DispatchQueue.main.async {
            withAnimation(.easeInOut(duration: 0.2)) {
                proxy.scrollTo(id, anchor: .center)
            }
        }
    }

    private var searchRow: some View {
        HStack(spacing: 8) {
            Image(systemName: "magnifyingglass")
                .foregroundStyle(.secondary)
            TextField(SkillsPickerFormatter.formatSearchPrompt(sidebarID: resolvedSidebar), text: $query)
                .textFieldStyle(.plain)
                .focused($searchFocused)
                .onSubmit { activateSelection() }
                .onKeyPress(.upArrow) {
                    moveSelection(-1)
                    return .handled
                }
                .onKeyPress(.downArrow) {
                    moveSelection(1)
                    return .handled
                }
                .onKeyPress(.leftArrow) {
                    moveSidebar(-1)
                    return .handled
                }
                .onKeyPress(.rightArrow) {
                    moveSidebar(1)
                    return .handled
                }
                .onKeyPress(.escape) {
                    handleEscape()
                    return .handled
                }
                .accessibilityIdentifier("skills-picker-search")
        }
        .padding(.horizontal, 12)
        .padding(.vertical, 10)
    }

    private func spanText(_ spans: [FuzzySpan], fallback: String, caption: Bool) -> Text {
        let shown = SkillsPickerFormatter.displaySpans(spans, fallback: fallback)
        var out = Text("")
        for span in shown {
            var piece = Text(span.text)
            if span.matched {
                piece = piece.bold().foregroundStyle(Color.accentColor)
            } else if caption {
                piece = piece.foregroundStyle(.secondary)
            }
            out = out + piece
        }
        return out
    }

    private func focusSearch() {
        searchFocused = true
        DispatchQueue.main.async {
            searchFocused = true
        }
        scheduleReload(query)
    }

    private func scheduleReload(_ q: String) {
        searchTask?.cancel()
        searchTask = Task {
            let trimmed = q.trimmingCharacters(in: .whitespacesAndNewlines)
            if !trimmed.isEmpty {
                try? await Task.sleep(nanoseconds: SkillsPickerFormatter.searchDebounceNanoseconds)
            }
            guard !Task.isCancelled else { return }
            await reload(query: q)
        }
    }

    private var emptyTitle: String {
        if !query.trimmingCharacters(in: .whitespacesAndNewlines).isEmpty {
            return SkillsPickerFormatter.formatNoResults()
        }
        return SkillsPickerFormatter.formatEmptyTitle(sidebarID: resolvedSidebar)
    }

    private func handleEscape() {
        if !query.isEmpty {
            query = ""
            return
        }
        onDismiss()
    }

    private func moveSelection(_ delta: Int) {
        let list = items
        let current = selectedID.flatMap { id in list.firstIndex(where: { $0.id == id }) }
        guard let next = SkillsPickerFormatter.nextListSelectionIndex(
            count: list.count,
            current: current,
            delta: delta
        ) else { return }
        selectedID = list[next].id
    }

    private func moveSidebar(_ delta: Int) {
        let ids = [
            SkillsPickerFormatter.sidebarAll,
            SkillsPickerFormatter.sidebarSkills,
            SkillsPickerFormatter.sidebarTemplates,
            SkillsPickerFormatter.sidebarFiles,
        ]
        let current = ids.firstIndex(of: resolvedSidebar) ?? 0
        var next = current + delta
        if next < 0 { next = 0 }
        if next >= ids.count { next = ids.count - 1 }
        sidebarID = ids[next]
    }

    private func activateSelection() {
        guard let id = selectedID, let item = items.first(where: { $0.id == id }) else {
            if let first = items.first {
                pick(first)
            }
            return
        }
        pick(item)
    }

    private func pick(_ item: InsertPickerItem) {
        let text = item.clipboardText
        NSPasteboard.general.clearContents()
        NSPasteboard.general.setString(text, forType: .string)
        let path = item.path
        let kind = item.kind
        onDismiss()
        CopiedToastController.shared.show()
        Task {
            switch kind {
            case .skill:
                try? await ServerClient.shared.recordSkillUse(path: path)
            case .template:
                try? await ServerClient.shared.recordTemplateUse(path: path)
            case .file:
                try? await ServerClient.shared.recordFileUse(path: path)
            }
        }
    }

    private func reload(query: String) async {
        loading = items.isEmpty
        errorText = nil
        let sidebar = resolvedSidebar
        do {
            let next: [InsertPickerItem]
            switch sidebar {
            case SkillsPickerFormatter.sidebarSkills:
                let resp = try await ServerClient.shared.listSkills(query: query)
                next = resp.skills.map { InsertPickerItem(kind: .skill, skill: $0, template: nil, file: nil) }
            case SkillsPickerFormatter.sidebarTemplates:
                let resp = try await ServerClient.shared.listTemplates(query: query)
                next = resp.templates.map { InsertPickerItem(kind: .template, skill: nil, template: $0, file: nil) }
            case SkillsPickerFormatter.sidebarFiles:
                let resp = try await ServerClient.shared.listFiles(query: query)
                next = resp.files.map { InsertPickerItem(kind: .file, skill: nil, template: nil, file: $0) }
            default:
                async let skillsResp = ServerClient.shared.listSkills(query: query)
                async let templatesResp = ServerClient.shared.listTemplates(query: query)
                async let filesResp = ServerClient.shared.listFiles(query: query)
                let skills = try await skillsResp
                let templates = try await templatesResp
                let files = try await filesResp
                next = SkillsPickerFormatter.mergeAllItems(
                    skills: skills.skills,
                    templates: templates.templates,
                    files: files.files
                )
            }
            guard !Task.isCancelled else { return }
            items = next
            if selectedID == nil || !items.contains(where: { $0.id == selectedID }) {
                selectedID = items.first?.id
            }
        } catch {
            if SkillsPickerFormatter.isIgnorableSearchError(error) || Task.isCancelled {
                return
            }
            errorText = error.localizedDescription
        }
        loading = false
        searchFocused = true
        DispatchQueue.main.async {
            searchFocused = true
        }
    }
}

@MainActor
final class CopiedToastController {
    static let shared = CopiedToastController()

    private var panel: NSPanel?
    private var hideWork: DispatchWorkItem?

    func show(message: String = SkillsPickerFormatter.formatCopiedToast()) {
        hideWork?.cancel()
        if panel == nil {
            let p = NSPanel(
                contentRect: NSRect(x: 0, y: 0, width: 160, height: 48),
                styleMask: [.borderless, .nonactivatingPanel],
                backing: .buffered,
                defer: false
            )
            p.isFloatingPanel = true
            p.level = .statusBar
            p.isOpaque = false
            p.backgroundColor = .clear
            p.hasShadow = true
            p.collectionBehavior = [.canJoinAllSpaces, .fullScreenAuxiliary, .ignoresCycle]
            p.isReleasedWhenClosed = false
            p.hidesOnDeactivate = false
            panel = p
        }
        let host = NSHostingView(rootView: CopiedToastView(text: message))
        host.frame = NSRect(x: 0, y: 0, width: 160, height: 48)
        panel?.contentView = host
        if let screen = NSScreen.main {
            let f = screen.visibleFrame
            panel?.setFrameOrigin(NSPoint(x: f.midX - 80, y: f.midY + 60))
        }
        panel?.orderFrontRegardless()
        let work = DispatchWorkItem { [weak self] in
            self?.panel?.orderOut(nil)
        }
        hideWork = work
        DispatchQueue.main.asyncAfter(deadline: .now() + 1.2, execute: work)
    }
}

struct CopiedToastView: View {
    let text: String

    var body: some View {
        Text(text)
            .font(.headline)
            .padding(.horizontal, 20)
            .padding(.vertical, 10)
            .background(.ultraThickMaterial, in: RoundedRectangle(cornerRadius: 12, style: .continuous))
            .accessibilityIdentifier("skills-copied-toast")
    }
}
