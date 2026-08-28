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
    @AppStorage(SkillsPickerFormatter.clipboardPathPrefixDefaultsKey) private var clipboardPathPrefix = ""
    @AppStorage(SkillsPickerFormatter.clipboardAppendOCRDefaultsKey) private var clipboardAppendOCR = false
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

    @State private var templateRoots: [TemplatesRootItem] = []
    @State private var showCreateTemplate = false
    @State private var showAddFile = false
    @State private var showAddCommand = false
    @State private var creatingBusy = false
    @State private var formError: String?

    @State private var newTemplateName = ""
    @State private var newTemplateDescription = ""
    @State private var newTemplateTags = ""
    @State private var newTemplateBody = ""
    @State private var newTemplateRootPath = ""

    @State private var addFilePath = ""
    @State private var addFileNote = ""

    @State private var addCommandText = ""
    @State private var addCommandNote = ""

    @State private var clipboardPeek: ClipboardPeekResponse?
    @State private var clipboardLoading = false
    @State private var clipboardBusy = false
    @State private var clipboardError: String?
    @State private var lastDumpPath: String = ""
    @State private var clipboardOCRText: String = ""
    @State private var clipboardOCRLoading = false
    @State private var clipboardOCRError: String?
    @State private var clipboardOCRTask: Task<Void, Never>?

    @State private var adhocText = ""
    @State private var adhocPath = ""
    @State private var adhocLoading = false
    @State private var adhocSaveStatus = ""
    @State private var adhocSaveTask: Task<Void, Never>?
    @State private var adhocDirty = false
    @State private var adhocSuppressChange = false
    @FocusState private var adhocFocused: Bool

    private var resolvedSidebar: String {
        SkillsPickerFormatter.normalizeSidebarID(sidebarID ?? storedSidebar)
    }

    private var isSearchableSidebar: Bool {
        SkillsPickerFormatter.isSearchableSidebar(resolvedSidebar)
    }

    private var showSidebarTitles: Bool {
        SkillsPickerFormatter.shouldShowSidebarTitles(width: Double(sidebarWidth))
    }

    private var showAddChrome: Bool {
        SkillsPickerFormatter.shouldShowAddButton(sidebarID: resolvedSidebar)
            && query.trimmingCharacters(in: .whitespacesAndNewlines).isEmpty
            && errorText == nil
            && !(loading && items.isEmpty)
    }

    var body: some View {
        ZStack {
            VStack(spacing: 0) {
                if isSearchableSidebar {
                    searchRow
                    Divider()
                }
                HStack(spacing: 0) {
                    sidebar
                        .frame(width: sidebarWidth)
                        .frame(maxHeight: .infinity)
                    sidebarResizeHandle
                    contentPane
                        .frame(minWidth: 280, maxWidth: .infinity, maxHeight: .infinity)
                }
            }
            .disabled(showCreateTemplate || showAddFile || showAddCommand)

            if showCreateTemplate {
                createTemplateOverlay
            }
            if showAddFile {
                addFileOverlay
            }
            if showAddCommand {
                addCommandOverlay
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
        .onDisappear {
            flushAdhocSave()
        }
        .onChange(of: query) { _, q in
            guard isSearchableSidebar else { return }
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
        .onChange(of: adhocText) { _, _ in
            guard resolvedSidebar == SkillsPickerFormatter.sidebarAdhoc else { return }
            guard !adhocLoading, !adhocSuppressChange else { return }
            adhocDirty = true
            adhocSaveStatus = SkillsPickerFormatter.formatAdhocDirtyStatus()
            scheduleAdhocSave()
        }
        .onExitCommand { handleEscape() }
        .onKeyPress(.upArrow) {
            guard !showCreateTemplate, !showAddFile, !showAddCommand, isSearchableSidebar else { return .ignored }
            moveSelection(-1)
            return .handled
        }
        .onKeyPress(.downArrow) {
            guard !showCreateTemplate, !showAddFile, !showAddCommand, isSearchableSidebar else { return .ignored }
            moveSelection(1)
            return .handled
        }
        .onKeyPress(.leftArrow) {
            guard !showCreateTemplate, !showAddFile, !showAddCommand else { return .ignored }
            moveSidebar(-1)
            return .handled
        }
        .onKeyPress(.rightArrow) {
            guard !showCreateTemplate, !showAddFile, !showAddCommand else { return .ignored }
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
            sidebarRow(id: SkillsPickerFormatter.sidebarCommands)
            // Draw the rule inside the Clipboard row — a bare Divider list row
            // gets sidebar min-height and shows as a large empty gap.
            sidebarRow(id: SkillsPickerFormatter.sidebarClipboard, separatorAbove: true)
            sidebarRow(id: SkillsPickerFormatter.sidebarAdhoc)
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

    private func sidebarRow(id: String, separatorAbove: Bool = false) -> some View {
        let title = SkillsPickerFormatter.formatSidebarTitle(id: id)
        let symbol = SkillsPickerFormatter.formatSidebarSymbol(id: id)
        return VStack(spacing: 0) {
            if separatorAbove {
                Rectangle()
                    .fill(Color(nsColor: .separatorColor))
                    .frame(height: 1)
                    .padding(.horizontal, 4)
                    .padding(.top, 2)
                    .padding(.bottom, 6)
                    .accessibilityIdentifier("insert-picker-sidebar-separator")
            }
            HStack(spacing: 8) {
                Image(systemName: symbol)
                    .frame(width: 20, alignment: .center)
                Text(title)
                    .lineLimit(1)
                    .opacity(showSidebarTitles ? 1 : 0)
                    .frame(maxWidth: showSidebarTitles ? .infinity : 0, alignment: .leading)
                    .clipped()
            }
            .frame(maxWidth: .infinity, alignment: .leading)
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
        switch resolvedSidebar {
        case SkillsPickerFormatter.sidebarClipboard:
            clipboardPane
        case SkillsPickerFormatter.sidebarAdhoc:
            adhocPane
        default:
            searchableContentPane
        }
    }

    @ViewBuilder
    private var searchableContentPane: some View {
        VStack(spacing: 0) {
            Group {
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
                                .multilineTextAlignment(.center)
                        }
                    }
                    .frame(maxWidth: .infinity, maxHeight: .infinity)
                } else {
                    ScrollViewReader { proxy in
                        List(items, selection: $selectedID) { item in
                            listRow(item)
                        }
                        .listStyle(.sidebar)
                        .scrollContentBackground(.hidden)
                        .onChange(of: selectedID) { _, id in
                            scrollList(proxy, to: id)
                        }
                    }
                }
            }
            .frame(maxWidth: .infinity, maxHeight: .infinity)

            if showAddChrome {
                Divider()
                addFooter
            }
        }
    }

    private var clipboardPane: some View {
        VStack(alignment: .leading, spacing: 12) {
            Text(SkillsPickerFormatter.formatClipboardHeading())
                .font(.headline)
            if clipboardLoading && clipboardPeek == nil {
                ProgressView()
                    .controlSize(.small)
                    .frame(maxWidth: .infinity, maxHeight: .infinity)
            } else {
                if let clipboardError {
                    Text(clipboardError)
                        .foregroundStyle(.red)
                        .font(.caption)
                }
                let peek = clipboardPeek ?? ClipboardPeekResponse()
                let kindLabel = SkillsPickerFormatter.formatClipboardKind(peek.kind)
                let canDump = SkillsPickerFormatter.canDumpClipboard(kind: peek.kind)
                let isImage = SkillsPickerFormatter.isClipboardImageKind(peek.kind)
                HStack(alignment: .firstTextBaseline, spacing: 8) {
                    Text(SkillsPickerFormatter.formatClipboardKindLabel())
                        .foregroundStyle(.secondary)
                    Text(kindLabel)
                    if !peek.ext.isEmpty, isImage {
                        Text("/ \(peek.ext)")
                            .foregroundStyle(.secondary)
                    }
                }
                .font(.callout)
                HStack(alignment: .firstTextBaseline, spacing: 8) {
                    Text(SkillsPickerFormatter.formatClipboardSizeLabel())
                        .foregroundStyle(.secondary)
                    Text(SkillsPickerFormatter.formatClipboardSize(bytes: peek.bytes))
                }
                .font(.callout)
                clipboardContentPreview(peek: peek, isImage: isImage)
                VStack(alignment: .leading, spacing: 4) {
                    Text(SkillsPickerFormatter.formatPathPrefixLabel())
                        .font(.caption)
                        .foregroundStyle(.secondary)
                    TextField(
                        SkillsPickerFormatter.formatPathPrefixPlaceholder(),
                        text: $clipboardPathPrefix
                    )
                    .textFieldStyle(.roundedBorder)
                    .font(.system(.body, design: .monospaced))
                    .accessibilityIdentifier("insert-picker-clipboard-path-prefix")
                    if isImage {
                        Toggle(isOn: $clipboardAppendOCR) {
                            Text(SkillsPickerFormatter.formatOCRCheckboxTitle())
                        }
                        .toggleStyle(.checkbox)
                        .help(SkillsPickerFormatter.formatOCRCheckboxHelp())
                        .accessibilityIdentifier("insert-picker-clipboard-append-ocr")
                        Text(SkillsPickerFormatter.formatOCRCheckboxHelp())
                            .font(.caption2)
                            .foregroundStyle(.secondary)
                    }
                }
                HStack(spacing: 12) {
                    Button(SkillsPickerFormatter.formatDumpToFileTitle()) {
                        Task { await dumpClipboard() }
                    }
                    .disabled(!canDump || clipboardBusy)
                    .accessibilityIdentifier("insert-picker-clipboard-dump")
                    Button(SkillsPickerFormatter.formatCopyFilePathTitle()) {
                        copyDumpPath()
                    }
                    .disabled(lastDumpPath.isEmpty || clipboardBusy)
                    .accessibilityIdentifier("insert-picker-clipboard-copy-path")
                    Spacer()
                    Button(SkillsPickerFormatter.formatRefreshTitle()) {
                        Task { await loadClipboardPeek() }
                    }
                    .disabled(clipboardBusy || clipboardLoading)
                    .accessibilityIdentifier("insert-picker-clipboard-refresh")
                }
                .buttonStyle(.borderless)
                if !lastDumpPath.isEmpty {
                    VStack(alignment: .leading, spacing: 2) {
                        Text(SkillsPickerFormatter.formatLastDumpLabel())
                            .font(.caption)
                            .foregroundStyle(.secondary)
                        Text(lastDumpPath)
                            .font(.system(.caption, design: .monospaced))
                            .textSelection(.enabled)
                            .lineLimit(2)
                    }
                }
                Spacer(minLength: 0)
            }
        }
        .padding(16)
        .frame(maxWidth: .infinity, maxHeight: .infinity, alignment: .topLeading)
        .accessibilityIdentifier("insert-picker-clipboard")
    }

    /// Shared preview box: plaintext shows peek text; image shows OCR (titled "OCR").
    @ViewBuilder
    private func clipboardContentPreview(peek: ClipboardPeekResponse, isImage: Bool) -> some View {
        if isImage {
            VStack(alignment: .leading, spacing: 4) {
                Text(SkillsPickerFormatter.formatOCRHeading())
                    .font(.caption)
                    .foregroundStyle(.secondary)
                clipboardPreviewBox {
                    if clipboardOCRLoading && clipboardOCRText.isEmpty && clipboardOCRError == nil {
                        HStack(spacing: 8) {
                            ProgressView()
                                .controlSize(.small)
                            Text(SkillsPickerFormatter.formatOCRRecognizing())
                                .font(.caption)
                                .foregroundStyle(.secondary)
                        }
                        .frame(maxWidth: .infinity, alignment: .leading)
                    } else if let clipboardOCRError {
                        Text(clipboardOCRError)
                            .font(.caption)
                            .foregroundStyle(.red)
                            .frame(maxWidth: .infinity, alignment: .leading)
                            .textSelection(.enabled)
                    } else if clipboardOCRText.isEmpty {
                        Text(SkillsPickerFormatter.formatOCREmpty())
                            .font(.caption)
                            .foregroundStyle(.secondary)
                            .frame(maxWidth: .infinity, alignment: .leading)
                    } else {
                        Text(clipboardOCRText)
                            .font(.system(.caption, design: .monospaced))
                            .frame(maxWidth: .infinity, alignment: .leading)
                            .textSelection(.enabled)
                    }
                }
                .accessibilityIdentifier("insert-picker-clipboard-ocr")
            }
        } else if !peek.preview.isEmpty {
            clipboardPreviewBox {
                Text(peek.preview)
                    .font(.system(.caption, design: .monospaced))
                    .frame(maxWidth: .infinity, alignment: .leading)
                    .textSelection(.enabled)
            }
            .accessibilityIdentifier("insert-picker-clipboard-preview")
        } else if peek.kind.lowercased() == "empty" {
            Text(SkillsPickerFormatter.formatEmptyHint(sidebarID: SkillsPickerFormatter.sidebarClipboard))
                .font(.caption)
                .foregroundStyle(.secondary)
        }
    }

    private func clipboardPreviewBox<Content: View>(@ViewBuilder content: () -> Content) -> some View {
        ScrollView {
            content()
        }
        .frame(maxHeight: 140)
        .padding(8)
        .background(Color(nsColor: .textBackgroundColor))
        .clipShape(RoundedRectangle(cornerRadius: 6))
    }

    private var adhocPane: some View {
        VStack(alignment: .leading, spacing: 10) {
            HStack {
                Text(SkillsPickerFormatter.formatAdhocHeading())
                    .font(.headline)
                Spacer()
                if !adhocSaveStatus.isEmpty {
                    Text(adhocSaveStatus)
                        .font(.caption)
                        .foregroundStyle(.secondary)
                        .accessibilityIdentifier("insert-picker-adhoc-status")
                }
            }
            if adhocLoading && adhocText.isEmpty {
                ProgressView()
                    .controlSize(.small)
                    .frame(maxWidth: .infinity, maxHeight: .infinity)
            } else {
                TextEditor(text: $adhocText)
                    .font(.system(.body, design: .monospaced))
                    .focused($adhocFocused)
                    .scrollContentBackground(.hidden)
                    .padding(6)
                    .background(Color(nsColor: .textBackgroundColor))
                    .clipShape(RoundedRectangle(cornerRadius: 6))
                    .accessibilityIdentifier("insert-picker-adhoc-editor")
                HStack {
                    Button(SkillsPickerFormatter.formatCopyTextTitle()) {
                        copyAdhocText()
                    }
                    .buttonStyle(.borderless)
                    .disabled(adhocText.isEmpty)
                    .accessibilityIdentifier("insert-picker-adhoc-copy")
                    Spacer()
                }
            }
        }
        .padding(16)
        .frame(maxWidth: .infinity, maxHeight: .infinity, alignment: .topLeading)
        .accessibilityIdentifier("insert-picker-adhoc")
    }

    private func listRow(_ item: InsertPickerItem) -> some View {
        HStack(alignment: .top, spacing: 8) {
            if resolvedSidebar == SkillsPickerFormatter.sidebarAll {
                Text(SkillsPickerFormatter.formatKindBadge(item.kind))
                    .font(.caption2.weight(.semibold))
                    .foregroundStyle(.secondary)
                    .frame(width: 64, alignment: .leading)
            }
            VStack(alignment: .leading, spacing: 2) {
                HStack(alignment: .firstTextBaseline, spacing: 8) {
                    spanText(
                        item.titleSpans,
                        fallback: item.title,
                        caption: false
                    )
                    .lineLimit(1)
                    .layoutPriority(1)
                    Spacer(minLength: 8)
                    if item.kind == .template {
                        let desc = item.trailingDescription
                        if !desc.isEmpty {
                            Text(desc)
                                .font(.caption)
                                .foregroundStyle(.secondary)
                                .lineLimit(1)
                                .truncationMode(.tail)
                        }
                    }
                    let count = SkillsPickerFormatter.formatUseCount(item.useCount)
                    if !count.isEmpty {
                        Text(count)
                            .font(.caption.monospacedDigit())
                            .foregroundStyle(.secondary)
                    }
                }
                if item.kind == .template {
                    spanText(
                        item.bodySpans,
                        fallback: item.subtitle,
                        caption: true
                    )
                    .font(.caption)
                    .lineLimit(1)
                } else {
                    spanText(
                        item.pathSpans,
                        fallback: item.subtitle,
                        caption: true
                    )
                    .font(.caption)
                    .lineLimit(1)
                }
            }
        }
        .tag(item.id)
        .id(item.id)
        .contentShape(Rectangle())
        .onTapGesture { pick(item) }
    }

    private var addFooter: some View {
        HStack {
            Button {
                beginAdd()
            } label: {
                Label(
                    SkillsPickerFormatter.formatAddButtonTitle(sidebarID: resolvedSidebar),
                    systemImage: "plus"
                )
            }
            .buttonStyle(.borderless)
            .accessibilityIdentifier("insert-picker-add")
            Spacer()
        }
        .padding(.horizontal, 12)
        .padding(.vertical, 8)
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
        if isSearchableSidebar {
            searchFocused = true
            DispatchQueue.main.async {
                searchFocused = true
            }
        }
        scheduleReload(query)
    }

    private func scheduleReload(_ q: String) {
        searchTask?.cancel()
        let sidebar = resolvedSidebar
        if sidebar == SkillsPickerFormatter.sidebarClipboard {
            Task { await loadClipboardPeek() }
            return
        }
        if sidebar == SkillsPickerFormatter.sidebarAdhoc {
            Task { await loadAdhocText() }
            return
        }
        searchTask = Task {
            let trimmed = q.trimmingCharacters(in: .whitespacesAndNewlines)
            if !trimmed.isEmpty {
                try? await Task.sleep(nanoseconds: SkillsPickerFormatter.searchDebounceNanoseconds)
            }
            guard !Task.isCancelled else { return }
            await reload(query: q)
        }
    }

    private func loadClipboardPeek() async {
        clipboardLoading = true
        clipboardError = nil
        defer { clipboardLoading = false }
        do {
            let peek = try await ServerClient.shared.peekClipboard()
            clipboardPeek = peek
            if SkillsPickerFormatter.isClipboardImageKind(peek.kind) {
                startClipboardOCR()
            } else {
                resetClipboardOCR()
            }
        } catch {
            clipboardError = error.localizedDescription
            resetClipboardOCR()
        }
    }

    private func dumpClipboard() async {
        clipboardBusy = true
        clipboardError = nil
        defer { clipboardBusy = false }
        do {
            let dumped = try await ServerClient.shared.dumpClipboard()
            lastDumpPath = dumped.path
            if SkillsPickerFormatter.isClipboardImageKind(dumped.kind),
               clipboardOCRText.isEmpty, !clipboardOCRLoading
            {
                startClipboardOCR(filePath: dumped.path)
            }
        } catch {
            clipboardError = error.localizedDescription
        }
    }

    private func copyDumpPath() {
        let text = SkillsPickerFormatter.formatClipboardCopyText(
            prefix: clipboardPathPrefix,
            path: lastDumpPath,
            appendOCR: clipboardAppendOCR,
            ocrText: clipboardOCRText
        )
        guard !text.isEmpty else { return }
        NSPasteboard.general.clearContents()
        NSPasteboard.general.setString(text, forType: .string)
        CopiedToastController.shared.show(message: SkillsPickerFormatter.formatPathCopiedToast())
    }

    private func resetClipboardOCR() {
        clipboardOCRTask?.cancel()
        clipboardOCRTask = nil
        clipboardOCRText = ""
        clipboardOCRError = nil
        clipboardOCRLoading = false
    }

    /// Run Vision OCR from pasteboard (default) or an already-dumped file path.
    private func startClipboardOCR(filePath: String? = nil) {
        clipboardOCRTask?.cancel()
        clipboardOCRLoading = true
        clipboardOCRError = nil
        let path = filePath
        clipboardOCRTask = Task.detached(priority: .userInitiated) {
            let result: Result<String, Error>
            do {
                let text: String
                if let path, !path.isEmpty {
                    text = try ClipboardOCR.recognize(filePath: path)
                } else {
                    text = try ClipboardOCR.recognizeFromPasteboard()
                }
                result = .success(text)
            } catch {
                result = .failure(error)
            }
            await MainActor.run {
                guard !Task.isCancelled else { return }
                clipboardOCRLoading = false
                switch result {
                case .success(let text):
                    clipboardOCRText = text
                    clipboardOCRError = nil
                case .failure(let error):
                    clipboardOCRText = ""
                    clipboardOCRError = error.localizedDescription
                }
            }
        }
    }

    private func loadAdhocText() async {
        adhocLoading = true
        defer { adhocLoading = false }
        do {
            let resp = try await ServerClient.shared.getAdhocText()
            adhocSuppressChange = true
            adhocText = resp.content
            adhocPath = resp.path
            adhocDirty = false
            adhocSaveStatus = SkillsPickerFormatter.formatAdhocSavedStatus()
            DispatchQueue.main.async {
                adhocSuppressChange = false
                adhocFocused = true
            }
        } catch {
            errorText = error.localizedDescription
            adhocSaveStatus = ""
        }
    }

    private func scheduleAdhocSave() {
        adhocSaveTask?.cancel()
        adhocSaveTask = Task {
            try? await Task.sleep(nanoseconds: SkillsPickerFormatter.adhocSaveDebounceNanoseconds)
            guard !Task.isCancelled else { return }
            await saveAdhocText()
        }
    }

    private func flushAdhocSave() {
        adhocSaveTask?.cancel()
        adhocSaveTask = nil
        guard adhocDirty else { return }
        let content = adhocText
        Task {
            do {
                _ = try await ServerClient.shared.putAdhocText(content: content)
            } catch {
                // best-effort on dismiss
            }
        }
        adhocDirty = false
    }

    private func saveAdhocText() async {
        adhocSaveStatus = SkillsPickerFormatter.formatAdhocSavingStatus()
        do {
            let resp = try await ServerClient.shared.putAdhocText(content: adhocText)
            adhocPath = resp.path
            adhocDirty = false
            adhocSaveStatus = SkillsPickerFormatter.formatAdhocSavedStatus()
        } catch {
            adhocSaveStatus = error.localizedDescription
        }
    }

    private func copyAdhocText() {
        NSPasteboard.general.clearContents()
        NSPasteboard.general.setString(adhocText, forType: .string)
        CopiedToastController.shared.show()
    }

    private var emptyTitle: String {
        if !query.trimmingCharacters(in: .whitespacesAndNewlines).isEmpty {
            return SkillsPickerFormatter.formatNoResults()
        }
        return SkillsPickerFormatter.formatEmptyTitle(sidebarID: resolvedSidebar)
    }

    private func handleEscape() {
        if showCreateTemplate {
            showCreateTemplate = false
            formError = nil
            return
        }
        if showAddFile {
            showAddFile = false
            formError = nil
            return
        }
        if showAddCommand {
            showAddCommand = false
            formError = nil
            return
        }
        if isSearchableSidebar, !query.isEmpty {
            query = ""
            return
        }
        flushAdhocSave()
        onDismiss()
    }

    private var createTemplateOverlay: some View {
        ZStack {
            Color.black.opacity(0.28)
                .ignoresSafeArea()
                .onTapGesture {
                    if !creatingBusy {
                        showCreateTemplate = false
                        formError = nil
                    }
                }
            VStack(alignment: .leading, spacing: 10) {
                Text(SkillsPickerFormatter.formatNewTemplateSheetTitle())
                    .font(.headline)
                TextField("Name", text: $newTemplateName)
                    .textFieldStyle(.roundedBorder)
                TextField("Description (optional)", text: $newTemplateDescription)
                    .textFieldStyle(.roundedBorder)
                TextField("Tags (comma-separated)", text: $newTemplateTags)
                    .textFieldStyle(.roundedBorder)
                if templateRoots.count > 1 {
                    Picker("Root", selection: $newTemplateRootPath) {
                        ForEach(templateRoots) { root in
                            Text(root.path).tag(root.path)
                        }
                    }
                } else if let only = templateRoots.first {
                    Text(only.path)
                        .font(.caption)
                        .foregroundStyle(.secondary)
                        .lineLimit(1)
                }
                Text("Body")
                    .font(.caption)
                    .foregroundStyle(.secondary)
                TextEditor(text: $newTemplateBody)
                    .font(.body)
                    .frame(minHeight: 100, maxHeight: 140)
                    .border(Color(nsColor: .separatorColor), width: 1)
                if let formError {
                    Text(formError)
                        .font(.caption)
                        .foregroundStyle(.red)
                }
                HStack {
                    Spacer()
                    Button("Cancel") {
                        showCreateTemplate = false
                        formError = nil
                    }
                    .keyboardShortcut(.cancelAction)
                    .disabled(creatingBusy)
                    Button("Create") {
                        Task { await submitCreateTemplate() }
                    }
                    .keyboardShortcut(.defaultAction)
                    .disabled(creatingBusy)
                }
            }
            .padding(16)
            .frame(width: 420)
            .background(Color(nsColor: .windowBackgroundColor))
            .clipShape(RoundedRectangle(cornerRadius: 10))
            .shadow(radius: 12)
        }
        .accessibilityIdentifier("insert-picker-create-template")
    }

    private var addFileOverlay: some View {
        ZStack {
            Color.black.opacity(0.28)
                .ignoresSafeArea()
                .onTapGesture {
                    if !creatingBusy {
                        showAddFile = false
                        formError = nil
                    }
                }
            VStack(alignment: .leading, spacing: 10) {
                Text(SkillsPickerFormatter.formatAddFileSheetTitle())
                    .font(.headline)
                HStack(spacing: 8) {
                    TextField("Path", text: $addFilePath)
                        .textFieldStyle(.roundedBorder)
                    Button("Browse…") {
                        browseAddFilePath()
                    }
                }
                TextField("Note (optional)", text: $addFileNote)
                    .textFieldStyle(.roundedBorder)
                if let formError {
                    Text(formError)
                        .font(.caption)
                        .foregroundStyle(.red)
                }
                HStack {
                    Spacer()
                    Button("Cancel") {
                        showAddFile = false
                        formError = nil
                    }
                    .keyboardShortcut(.cancelAction)
                    .disabled(creatingBusy)
                    Button("Add") {
                        Task { await submitAddFile() }
                    }
                    .keyboardShortcut(.defaultAction)
                    .disabled(creatingBusy)
                }
            }
            .padding(16)
            .frame(width: 420)
            .background(Color(nsColor: .windowBackgroundColor))
            .clipShape(RoundedRectangle(cornerRadius: 10))
            .shadow(radius: 12)
        }
        .accessibilityIdentifier("insert-picker-add-file")
    }

    private var addCommandOverlay: some View {
        ZStack {
            Color.black.opacity(0.28)
                .ignoresSafeArea()
                .onTapGesture {
                    if !creatingBusy {
                        showAddCommand = false
                        formError = nil
                    }
                }
            VStack(alignment: .leading, spacing: 10) {
                Text(SkillsPickerFormatter.formatAddCommandSheetTitle())
                    .font(.headline)
                TextField("Command", text: $addCommandText)
                    .textFieldStyle(.roundedBorder)
                TextField("Note (optional)", text: $addCommandNote)
                    .textFieldStyle(.roundedBorder)
                if let formError {
                    Text(formError)
                        .font(.caption)
                        .foregroundStyle(.red)
                }
                HStack {
                    Spacer()
                    Button("Cancel") {
                        showAddCommand = false
                        formError = nil
                    }
                    .keyboardShortcut(.cancelAction)
                    .disabled(creatingBusy)
                    Button("Add") {
                        Task { await submitAddCommand() }
                    }
                    .keyboardShortcut(.defaultAction)
                    .disabled(creatingBusy)
                }
            }
            .padding(16)
            .frame(width: 420)
            .background(Color(nsColor: .windowBackgroundColor))
            .clipShape(RoundedRectangle(cornerRadius: 10))
            .shadow(radius: 12)
        }
        .accessibilityIdentifier("insert-picker-add-command")
    }

    private func beginAdd() {
        formError = nil
        switch resolvedSidebar {
        case SkillsPickerFormatter.sidebarTemplates:
            beginAddTemplate()
        case SkillsPickerFormatter.sidebarFiles:
            addFilePath = ""
            addFileNote = ""
            showAddFile = true
        case SkillsPickerFormatter.sidebarCommands:
            addCommandText = ""
            addCommandNote = ""
            showAddCommand = true
        default:
            break
        }
    }

    private func beginAddTemplate() {
        if templateRoots.isEmpty {
            Task { await chooseAndRegisterTemplateRoot(thenCreate: true) }
            return
        }
        resetCreateTemplateForm()
        showCreateTemplate = true
    }

    private func resetCreateTemplateForm() {
        newTemplateName = ""
        newTemplateDescription = ""
        newTemplateTags = ""
        newTemplateBody = ""
        newTemplateRootPath = templateRoots.first?.path ?? ""
        formError = nil
    }

    private func browseAddFilePath() {
        let panel = NSOpenPanel()
        panel.canChooseFiles = true
        panel.canChooseDirectories = true
        panel.allowsMultipleSelection = false
        panel.canCreateDirectories = false
        panel.prompt = "Choose"
        panel.message = SkillsPickerFormatter.formatAddFileSheetTitle()
        guard panel.runModal() == .OK, let url = panel.url else { return }
        addFilePath = url.path
    }

    private func chooseAndRegisterTemplateRoot(thenCreate: Bool) async {
        let panel = NSOpenPanel()
        panel.canChooseFiles = false
        panel.canChooseDirectories = true
        panel.allowsMultipleSelection = false
        panel.canCreateDirectories = true
        panel.prompt = "Choose"
        panel.message = SkillsPickerFormatter.formatChooseTemplateFolderTitle()
        guard panel.runModal() == .OK, let url = panel.url else { return }
        creatingBusy = true
        formError = nil
        defer { creatingBusy = false }
        do {
            let resp = try await ServerClient.shared.addTemplateDir(path: url.path)
            if !templateRoots.contains(where: { $0.path == resp.root.path }) {
                templateRoots.append(resp.root)
            }
            await reload(query: query)
            if thenCreate {
                resetCreateTemplateForm()
                newTemplateRootPath = resp.root.path
                showCreateTemplate = true
            }
        } catch {
            formError = error.localizedDescription
            errorText = error.localizedDescription
        }
    }

    private func submitCreateTemplate() async {
        let name = newTemplateName.trimmingCharacters(in: .whitespacesAndNewlines)
        let body = newTemplateBody
        guard !name.isEmpty else {
            formError = "name is required"
            return
        }
        guard !body.trimmingCharacters(in: .whitespacesAndNewlines).isEmpty else {
            formError = "body is required"
            return
        }
        if templateRoots.isEmpty {
            formError = "no template roots registered"
            return
        }
        creatingBusy = true
        formError = nil
        defer { creatingBusy = false }
        let tags = newTemplateTags
            .split(separator: ",")
            .map { $0.trimmingCharacters(in: .whitespacesAndNewlines) }
            .filter { !$0.isEmpty }
        let root: String? = templateRoots.count > 1 ? newTemplateRootPath : nil
        do {
            let created = try await ServerClient.shared.createTemplate(
                name: name,
                body: body,
                description: newTemplateDescription,
                tags: tags,
                root: root
            )
            showCreateTemplate = false
            await reload(query: query)
            selectedID = InsertPickerItem(
                kind: .template,
                skill: nil,
                template: created.template,
                file: nil
            ).id
        } catch {
            formError = error.localizedDescription
        }
    }

    private func submitAddFile() async {
        let path = addFilePath.trimmingCharacters(in: .whitespacesAndNewlines)
        guard !path.isEmpty else {
            formError = "path is required"
            return
        }
        creatingBusy = true
        formError = nil
        defer { creatingBusy = false }
        let note = addFileNote.trimmingCharacters(in: .whitespacesAndNewlines)
        do {
            let added = try await ServerClient.shared.addFile(
                path: path,
                note: note.isEmpty ? nil : note
            )
            showAddFile = false
            await reload(query: query)
            selectedID = InsertPickerItem(
                kind: .file,
                skill: nil,
                template: nil,
                file: added.file
            ).id
        } catch {
            formError = error.localizedDescription
        }
    }

    private func submitAddCommand() async {
        let command = addCommandText.trimmingCharacters(in: .whitespacesAndNewlines)
        guard !command.isEmpty else {
            formError = "command is required"
            return
        }
        creatingBusy = true
        formError = nil
        defer { creatingBusy = false }
        let note = addCommandNote.trimmingCharacters(in: .whitespacesAndNewlines)
        do {
            let added = try await ServerClient.shared.addCommand(
                command: command,
                note: note.isEmpty ? nil : note
            )
            showAddCommand = false
            await reload(query: query)
            selectedID = InsertPickerItem(
                kind: .command,
                skill: nil,
                template: nil,
                file: nil,
                command: added.command
            ).id
        } catch {
            formError = error.localizedDescription
        }
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
        let ids = SkillsPickerFormatter.sidebarOrder
        let current = ids.firstIndex(of: resolvedSidebar) ?? 0
        var next = current + delta
        if next < 0 { next = 0 }
        if next >= ids.count { next = ids.count - 1 }
        // Flush pending adhoc edits when leaving that pane.
        if resolvedSidebar == SkillsPickerFormatter.sidebarAdhoc,
           ids[next] != SkillsPickerFormatter.sidebarAdhoc {
            flushAdhocSave()
        }
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
            case .command:
                try? await ServerClient.shared.recordCommandUse(command: path)
            }
        }
    }

    private func reload(query: String) async {
        let sidebar = resolvedSidebar
        if sidebar == SkillsPickerFormatter.sidebarClipboard {
            await loadClipboardPeek()
            return
        }
        if sidebar == SkillsPickerFormatter.sidebarAdhoc {
            await loadAdhocText()
            return
        }
        loading = items.isEmpty
        errorText = nil
        do {
            let next: [InsertPickerItem]
            switch sidebar {
            case SkillsPickerFormatter.sidebarSkills:
                let resp = try await ServerClient.shared.listSkills(query: query)
                next = resp.skills.map { InsertPickerItem(kind: .skill, skill: $0, template: nil, file: nil) }
            case SkillsPickerFormatter.sidebarTemplates:
                let resp = try await ServerClient.shared.listTemplates(query: query)
                templateRoots = resp.roots
                next = resp.templates.map { InsertPickerItem(kind: .template, skill: nil, template: $0, file: nil) }
            case SkillsPickerFormatter.sidebarFiles:
                let resp = try await ServerClient.shared.listFiles(query: query)
                next = resp.files.map { InsertPickerItem(kind: .file, skill: nil, template: nil, file: $0) }
            case SkillsPickerFormatter.sidebarCommands:
                let resp = try await ServerClient.shared.listCommands(query: query)
                next = resp.commands.map {
                    InsertPickerItem(kind: .command, skill: nil, template: nil, file: nil, command: $0)
                }
            default:
                async let skillsResp = ServerClient.shared.listSkills(query: query)
                async let templatesResp = ServerClient.shared.listTemplates(query: query)
                async let filesResp = ServerClient.shared.listFiles(query: query)
                async let commandsResp = ServerClient.shared.listCommands(query: query)
                let skills = try await skillsResp
                let templates = try await templatesResp
                let files = try await filesResp
                let commands = try await commandsResp
                templateRoots = templates.roots
                next = SkillsPickerFormatter.mergeAllItems(
                    skills: skills.skills,
                    templates: templates.templates,
                    files: files.files,
                    commands: commands.commands
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
        if !showCreateTemplate && !showAddFile && !showAddCommand && SkillsPickerFormatter.isSearchableSidebar(sidebar) {
            searchFocused = true
            DispatchQueue.main.async {
                searchFocused = true
            }
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
