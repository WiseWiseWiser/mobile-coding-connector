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
        if let panel {
            let host = NSHostingView(rootView: view)
            host.frame = NSRect(x: 0, y: 0, width: 520, height: 400)
            panel.contentView = host
            hosting = host
            panel.collectionBehavior = [.canJoinAllSpaces, .fullScreenAuxiliary]
            position(panel)
            panel.makeKeyAndOrderFront(nil)
            startClickOutsideMonitor()
            return
        }
        let panel = NSPanel(
            contentRect: NSRect(x: 0, y: 0, width: 520, height: 400),
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
        host.frame = NSRect(x: 0, y: 0, width: 520, height: 400)
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

    @State private var query = ""
    @State private var skills: [SkillsPickerItem] = []
    @State private var selectedID: String?
    @State private var loading = true
    @State private var errorText: String?
    @State private var searchTask: Task<Void, Never>?
    @FocusState private var searchFocused: Bool

    var body: some View {
        VStack(spacing: 0) {
            searchRow
            Divider()
            if let errorText {
                Text(errorText)
                    .foregroundStyle(.red)
                    .font(.caption)
                    .padding(8)
            }
            if loading && skills.isEmpty {
                ProgressView()
                    .controlSize(.small)
                    .frame(maxWidth: .infinity, maxHeight: .infinity)
            } else if visible.isEmpty {
                VStack(spacing: 6) {
                    Text(emptyTitle)
                        .font(.headline)
                    if query.trimmingCharacters(in: .whitespacesAndNewlines).isEmpty {
                        Text(SkillsPickerFormatter.formatEmptyHint())
                            .font(.caption)
                            .foregroundStyle(.secondary)
                    }
                }
                .frame(maxWidth: .infinity, maxHeight: .infinity)
            } else {
                List(visible, selection: $selectedID) { skill in
                    HStack {
                        VStack(alignment: .leading, spacing: 2) {
                            spanText(
                                skill.titleSpans,
                                fallback: SkillsPickerFormatter.formatTitle(skill),
                                caption: false
                            )
                            .lineLimit(1)
                            spanText(
                                skill.pathSpans,
                                fallback: SkillsPickerFormatter.formatSubtitle(skill),
                                caption: true
                            )
                            .font(.caption)
                            .lineLimit(1)
                        }
                        Spacer()
                        let count = SkillsPickerFormatter.formatUseCount(skill.useCount)
                        if !count.isEmpty {
                            Text(count)
                                .font(.caption.monospacedDigit())
                                .foregroundStyle(.secondary)
                        }
                    }
                    .tag(skill.path)
                    .contentShape(Rectangle())
                    .onTapGesture { pick(skill) }
                }
                .listStyle(.sidebar)
                .scrollContentBackground(.hidden)
            }
        }
        .frame(width: 520, height: 400)
        .defaultFocus($searchFocused, true)
        .accessibilityIdentifier("skills-picker")
        .onAppear { focusSearch() }
        .onChange(of: query) { _, q in
            scheduleReload(q)
        }
        .onChange(of: visible.map(\.path)) { _, ids in
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
        .onKeyPress(.escape) {
            handleEscape()
            return .handled
        }
    }

    private var searchRow: some View {
        HStack(spacing: 8) {
            Image(systemName: "magnifyingglass")
                .foregroundStyle(.secondary)
            TextField(SkillsPickerFormatter.formatSearchPrompt(), text: $query)
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

    private var visible: [SkillsPickerItem] {
        skills
    }

    private var emptyTitle: String {
        if !query.trimmingCharacters(in: .whitespacesAndNewlines).isEmpty {
            return SkillsPickerFormatter.formatNoResults()
        }
        return SkillsPickerFormatter.formatEmptyTitle()
    }

    private func handleEscape() {
        if !query.isEmpty {
            query = ""
            return
        }
        onDismiss()
    }

    private func moveSelection(_ delta: Int) {
        let items = visible
        guard !items.isEmpty else { return }
        let current = items.firstIndex(where: { $0.path == selectedID }) ?? (delta > 0 ? -1 : 0)
        var next = current + delta
        if next < 0 { next = 0 }
        if next >= items.count { next = items.count - 1 }
        selectedID = items[next].path
    }

    private func activateSelection() {
        guard let id = selectedID, let skill = visible.first(where: { $0.path == id }) else {
            if let first = visible.first {
                pick(first)
            }
            return
        }
        pick(skill)
    }

    private func pick(_ skill: SkillsPickerItem) {
        NSPasteboard.general.clearContents()
        NSPasteboard.general.setString(skill.path, forType: .string)
        let path = skill.path
        onDismiss()
        CopiedToastController.shared.show()
        Task {
            try? await ServerClient.shared.recordSkillUse(path: path)
        }
    }

    private func reload(query: String) async {
        loading = skills.isEmpty
        errorText = nil
        do {
            let resp = try await ServerClient.shared.listSkills(query: query)
            guard !Task.isCancelled else { return }
            skills = resp.skills
            if selectedID == nil || !skills.contains(where: { $0.path == selectedID }) {
                selectedID = skills.first?.path
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
