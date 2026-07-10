import SwiftUI
import ServiceManagement
import AppKit
import AICriticMacShared

/// Remote menu-bar app entry (ai-critic-remote-macos / AI Critic(Remote)).
/// Does not spawn a local keep-alive daemon; daemon restart actions are omitted.
@MainActor
final class RemoteAppState: ObservableObject {
    @Published var menuLabel = "Remote …"
    @Published var statusLine = "Not configured — open Configure… to add a remote server"
    @Published var serverURL: String = ""
    @Published var hasEndpoint = false
    @Published var token: String = ""
    @Published var services: [ServiceStatus] = []
    @Published var cronTasks: [CronTaskStatus] = []
    @Published var terminals: [TerminalSession] = []
    @Published var domains: [RemoteDomain] = []
    @Published var defaultServer: String = ""
    @AppStorage("menuBarDisplayMode") var menuBarDisplayMode = "rotating"

    // MARK: - Periodic machine backup (default OFF; per active Server)

    /// Explicit default: periodic backup is not auto-enabled on launch.
    @Published var backupEnabled: Bool = false
    @Published var backupPhase: BackupMenuFormatter.Phase = .off
    @Published var backupLastFinishedAt: Date?
    @Published var backupNextRunAt: Date?
    @Published var backupLastError: String = ""
    @Published var backupRunning: Bool = false
    @Published var backupRecent: [BackupMenuFormatter.FileEntry] = []

    /// Shared HTTP client for remote service/cron APIs (base URL + Bearer token).
    /// Uses configured server baseURL + Authorization Bearer — not keep-alive port 23312.
    let serviceClient = ServiceClient()
    /// Stream + archive_token download client for machine backup.
    let machineBackupClient = MachineBackupClient()

    /// Override for tests; empty means use default CLI path.
    var configPathOverride: String?

    private let isRemoteApp = true
    private var backupTickTask: Task<Void, Never>?

    func configPath() -> String {
        if let override = configPathOverride, !override.isEmpty {
            return override
        }
        return RemoteConfigStore.defaultConfigPath()
    }

    /// Filesystem scope key for the active server URL.
    var backupServerName: String {
        BackupMenuFormatter.serverNameFromURL(serverURL)
    }

    var backupDirectory: String {
        BackupMenuFormatter.backupDirForServerURL(serverURL)
    }

    var backupStatusTitle: String {
        let st = BackupMenuFormatter.TaskStatus(
            enabled: backupEnabled,
            phase: backupRunning ? .running : (backupEnabled ? backupPhase : .off),
            lastFinishedAt: backupLastFinishedAt,
            nextRunAt: backupNextRunAt,
            lastError: backupLastError
        )
        return BackupMenuFormatter.formatBackupStatusTitle(st)
    }

    func refresh() async {
        let path = configPath()
        do {
            let result = try RemoteConfigStore.statusFromFile(path: path)
            statusLine = result.line
            serverURL = result.server
            hasEndpoint = result.resolved
            menuLabel = result.resolved ? "Remote" : "Remote …"

            if let cfg = try RemoteConfigStore.load(path: path) {
                domains = cfg.domains
                defaultServer = cfg.defaultServer
                if result.resolved {
                    let (ep, _) = RemoteConfigStore.resolve(cfg)
                    token = ep.token
                    serviceClient.configure(baseURL: ep.server, token: ep.token)
                    machineBackupClient.configure(baseURL: ep.server, token: ep.token)
                } else {
                    token = ""
                    serviceClient.configure(baseURL: "", token: "")
                    machineBackupClient.configure(baseURL: "", token: "")
                    services = []
                    cronTasks = []
                    terminals = []
                }
            } else {
                domains = []
                defaultServer = ""
                token = ""
                serviceClient.configure(baseURL: "", token: "")
                machineBackupClient.configure(baseURL: "", token: "")
                services = []
                cronTasks = []
                terminals = []
            }

            if hasEndpoint {
                await refreshServices()
                await refreshCronTasks()
                await refreshTerminals()
            }
            reloadBackupStateFromDisk()
            startBackupTickLoopIfNeeded()
        } catch {
            statusLine = "Not configured — open Configure… to add a remote server"
            serverURL = ""
            hasEndpoint = false
            token = ""
            domains = []
            defaultServer = ""
            serviceClient.configure(baseURL: "", token: "")
            machineBackupClient.configure(baseURL: "", token: "")
            services = []
            cronTasks = []
            terminals = []
            menuLabel = "Remote …"
            reloadBackupStateFromDisk()
        }
    }

    /// Load per-server backup task state (default enabled=false — never force-enable on launch).
    func reloadBackupStateFromDisk() {
        let name = backupServerName
        guard !name.isEmpty else {
            backupEnabled = false
            backupPhase = .off
            backupLastFinishedAt = nil
            backupNextRunAt = nil
            backupLastError = ""
            backupRecent = []
            return
        }
        let st = PeriodicBackupStore.state(for: name)
        // Keep explicit default-off: PeriodicBackupServerState.enabled defaults to false.
        backupEnabled = st.enabled
        backupLastFinishedAt = PeriodicBackupStore.parseRFC3339(st.lastFinishedAt)
        backupNextRunAt = PeriodicBackupStore.parseRFC3339(st.nextRunAt)
        backupLastError = st.lastError
        if st.enabled {
            if st.lastStatus == "error" {
                backupPhase = .error
            } else {
                backupPhase = .idle
            }
        } else {
            backupPhase = .off
        }
        refreshBackupRecent()
    }

    func refreshBackupRecent() {
        let dir = backupDirectory
        guard !dir.isEmpty, !backupServerName.isEmpty else {
            backupRecent = []
            return
        }
        backupRecent = PeriodicBackupStore.listBackupEntries(dir: dir)
    }

    func setBackupEnabled(_ enable: Bool) async {
        let name = backupServerName
        guard !name.isEmpty else { return }
        let now = Date()
        let interval = TimeInterval(BackupMenuFormatter.backupIntervalSeconds)
        var shouldRunNow = false
        do {
            try PeriodicBackupStore.update(serverName: name) { st in
                st.enabled = enable
                if enable {
                    st.lastError = ""
                    st.lastStatus = "idle"
                    let last = PeriodicBackupStore.parseRFC3339(st.lastFinishedAt)
                    shouldRunNow = BackupMenuFormatter.shouldRunOnEnable(
                        lastFinished: last,
                        now: now,
                        interval: interval
                    )
                    if !shouldRunNow, let last {
                        st.nextRunAt = PeriodicBackupStore.formatRFC3339(last.addingTimeInterval(interval))
                    } else if shouldRunNow {
                        st.nextRunAt = PeriodicBackupStore.formatRFC3339(now)
                    }
                } else {
                    st.nextRunAt = ""
                }
            }
        } catch {
            // Keep prior state on disk write failure.
        }
        reloadBackupStateFromDisk()
        if enable && shouldRunNow {
            await runBackupNow(triggeredBySchedule: true)
        }
    }

    func runBackupNow(triggeredBySchedule: Bool = false) async {
        let name = backupServerName
        guard !name.isEmpty, hasEndpoint, machineBackupClient.isConfigured else { return }
        if backupRunning { return }

        backupRunning = true
        backupPhase = .running
        let now = Date()
        let dir = backupDirectory
        let filename = BackupMenuFormatter.backupArchiveFilename(utc: now)
        let dest = (dir as NSString).appendingPathComponent(filename)

        do {
            try PeriodicBackupStore.update(serverName: name) { st in
                st.lastStartedAt = PeriodicBackupStore.formatRFC3339(now)
                st.lastStatus = "running"
                st.lastError = ""
            }
        } catch {}

        do {
            // Stream path + archive_token download (same as CLI machine backup).
            let size = try await machineBackupClient.downloadBackupArchive(to: dest)
            let finished = Date()
            let next = finished.addingTimeInterval(TimeInterval(BackupMenuFormatter.backupIntervalSeconds))
            try PeriodicBackupStore.update(serverName: name) { st in
                st.lastFinishedAt = PeriodicBackupStore.formatRFC3339(finished)
                st.lastStatus = "idle"
                st.lastError = ""
                st.lastOutputPath = dest
                st.lastSizeBytes = size
                if st.enabled {
                    st.nextRunAt = PeriodicBackupStore.formatRFC3339(next)
                }
            }
            PeriodicBackupStore.pruneBackupDir(dir: dir, now: finished)
            backupRunning = false
            backupPhase = backupEnabled ? .idle : .off
            backupLastError = ""
            reloadBackupStateFromDisk()
        } catch {
            let finished = Date()
            let msg = error.localizedDescription
            try? PeriodicBackupStore.update(serverName: name) { st in
                st.lastFinishedAt = PeriodicBackupStore.formatRFC3339(finished)
                st.lastStatus = "error"
                st.lastError = msg
                if st.enabled {
                    st.nextRunAt = PeriodicBackupStore.formatRFC3339(
                        finished.addingTimeInterval(TimeInterval(BackupMenuFormatter.backupIntervalSeconds))
                    )
                }
            }
            backupRunning = false
            backupPhase = backupEnabled ? .error : .off
            backupLastError = msg
            reloadBackupStateFromDisk()
        }
    }

    func revealBackupInFinder() {
        let dir = backupDirectory
        guard !dir.isEmpty else { return }
        try? FileManager.default.createDirectory(atPath: dir, withIntermediateDirectories: true)
        NSWorkspace.shared.open(URL(fileURLWithPath: dir))
    }

    /// Timer while app running: check due schedule for the active server.
    func startBackupTickLoopIfNeeded() {
        guard backupTickTask == nil else { return }
        backupTickTask = Task { @MainActor in
            while !Task.isCancelled {
                // Check more often than the 1h interval so due runs are not delayed too long.
                try? await Task.sleep(nanoseconds: 30_000_000_000)
                await self.checkBackupDue()
            }
        }
    }

    func checkBackupDue() async {
        reloadBackupStateFromDisk()
        let interval = TimeInterval(BackupMenuFormatter.backupIntervalSeconds)
        _ = interval
        let due = BackupMenuFormatter.shouldRunDue(
            enabled: backupEnabled,
            running: backupRunning,
            nextRunAt: backupNextRunAt,
            now: Date()
        )
        if due {
            await runBackupNow(triggeredBySchedule: true)
        }
    }
}

    func refreshServices() async {
        guard serviceClient.isConfigured else {
            services = []
            return
        }
        do {
            services = try await serviceClient.listServices()
        } catch {
            // Keep prior list when server is temporarily unreachable.
        }
    }

    func refreshCronTasks() async {
        guard serviceClient.isConfigured else {
            cronTasks = []
            return
        }
        do {
            // listCronTasks uses serviceClient baseURL + Bearer token (/api/cron-tasks)
            cronTasks = try await serviceClient.listCronTasks()
        } catch {
            // Keep prior list when server is temporarily unreachable.
        }
    }

    func refreshTerminals() async {
        guard serviceClient.isConfigured else {
            terminals = []
            return
        }
        do {
            terminals = try await serviceClient.listTerminalSessions()
        } catch {
            // Keep prior list when server is temporarily unreachable.
        }
    }

    /// Persist selected domain as `default` and reload clients (services + terminals + browser).
    func selectDefaultDomain(server: String) async {
        let path = configPath()
        do {
            guard let cfg = try RemoteConfigStore.load(path: path) else { return }
            let updated = try RemoteConfigStore.selectDefaultDomain(cfg, serverURL: server)
            try RemoteConfigStore.save(path: path, config: updated)
            await refresh()
        } catch {
            // Keep prior selection on failure.
        }
    }

    func openAttachTerminal(sessionID: String) {
        let binary = TerminalMenuFormatter.agentBinaryForApp(isRemote: isRemoteApp)
        let cmd = TerminalMenuFormatter.buildTerminalAttachCommand(
            agentBinary: binary,
            sessionID: sessionID
        )
        ITermOpener.openCommandOrAlert(cmd)
    }

    func openNewTerminal() {
        let binary = TerminalMenuFormatter.agentBinaryForApp(isRemote: isRemoteApp)
        let cmd = TerminalMenuFormatter.buildTerminalNewCommand(agentBinary: binary)
        ITermOpener.openCommandOrAlert(cmd)
    }
}

class RemoteAppDelegate: NSObject, NSApplicationDelegate {
    weak var state: RemoteAppState?

    func applicationDidFinishLaunching(_ notification: Notification) {
        NSApp.setActivationPolicy(.accessory)
        Task { @MainActor in
            await state?.refresh()
            startRefreshLoop()
        }
    }

    func applicationShouldTerminateAfterLastWindowClosed(_ sender: NSApplication) -> Bool {
        false
    }

    /// Periodic services + cron + terminals refresh (PeriodicRefreshInterval = 30s).
    @MainActor
    private func startRefreshLoop() {
        Task { @MainActor in
            while !Task.isCancelled {
                try? await Task.sleep(nanoseconds: TerminalMenuFormatter.periodicRefreshIntervalNanoseconds)
                await state?.refresh() // includes listCronTasks / refreshCronTasks path
            }
        }
    }
}

@available(macOS 15.0, *)
@main
struct AICriticRemoteApp: App {
    @NSApplicationDelegateAdaptor(RemoteAppDelegate.self) var appDelegate
    @StateObject private var state = RemoteAppState()
    /// Launch-at-login for this remote menubar app (same as local Auto Start).
    @AppStorage("autoStart") private var autoStart = false

    init() {
        let state = RemoteAppState()
        _state = StateObject(wrappedValue: state)
        appDelegate.state = state
        _autoStart.wrappedValue = SMAppService.mainApp.status == .enabled
    }

    var body: some Scene {
        Window("Settings", id: "settings") {
            SettingsView(
                menuBarDisplayMode: $state.menuBarDisplayMode,
                showRemoteConnection: true,
                onConnectionSaved: {
                    Task { @MainActor in
                        await state.refresh()
                    }
                }
            )
        }
        .windowResizability(.contentSize)
        .defaultLaunchBehavior(.suppressed)

        MenuBarExtra {
            RemoteMenuBarDropdown(
                state: state,
                autoStart: $autoStart,
                showSettings: showSettingsWindow
            )
        } label: {
            Text(state.menuLabel)
        }
    }

    private func showSettingsWindow(openWindow: OpenWindowAction) {
        NSApp.setActivationPolicy(.regular)
        NSApp.activate(ignoringOtherApps: true)
        openWindow(id: "settings")
        if let window = NSApp.windows.first(where: { $0.title == "Settings" }) {
            window.makeKeyAndOrderFront(nil)
            return
        }
        Task { @MainActor in
            for _ in 0..<15 {
                openWindow(id: "settings")
                if let window = NSApp.windows.first(where: { $0.title == "Settings" }) {
                    window.makeKeyAndOrderFront(nil)
                    return
                }
                try? await Task.sleep(nanoseconds: 100_000_000)
            }
        }
    }
}

@available(macOS 15.0, *)
private struct RemoteMenuBarDropdown: View {
    @ObservedObject var state: RemoteAppState
    @Binding var autoStart: Bool
    @AppStorage("defaultBrowser") private var defaultBrowser = BrowserPreference.default.rawValue
    @Environment(\.openWindow) private var openWindow
    let showSettings: (OpenWindowAction) -> Void

    var body: some View {
        VStack(alignment: .leading, spacing: 6) {
            Text(state.statusLine)
                .font(.caption)
                .foregroundStyle(.secondary)
                .fixedSize(horizontal: false, vertical: true)

            Divider()

            // Level-1 Server domain switcher (remote only)
            Menu("Server") {
                if state.domains.isEmpty {
                    Text("No servers configured")
                } else {
                    ForEach(state.domains) { domain in
                        let isSelected = RemoteConfigStore.normalizeServer(domain.server)
                            == RemoteConfigStore.normalizeServer(state.defaultServer)
                            || (state.defaultServer.isEmpty
                                && RemoteConfigStore.normalizeServer(domain.server)
                                == RemoteConfigStore.normalizeServer(state.serverURL))
                        Button {
                            Task { await state.selectDefaultDomain(server: domain.server) }
                        } label: {
                            HStack {
                                Text(domain.server)
                                if isSelected {
                                    Spacer()
                                    Text("✓")
                                }
                            }
                        }
                    }
                }
            }
            .accessibilityIdentifier("server-switcher")

            Menu("Services") {
                if !state.hasEndpoint {
                    Text("Not configured")
                } else if state.services.isEmpty {
                    Text(ServiceMenuFormatter.formatServicesEmptyLabel())
                } else {
                    ForEach(state.services) { service in
                        Menu(ServiceMenuFormatter.formatServiceTitle(
                            name: service.name,
                            status: service.status,
                            enabled: service.enabled
                        )) {
                            Button("Start") {
                                Task {
                                    await runServiceAction {
                                        try await state.serviceClient.startService(id: service.id)
                                    }
                                }
                            }
                            Button("Restart") {
                                Task {
                                    await runServiceAction {
                                        try await state.serviceClient.restartService(id: service.id)
                                    }
                                }
                            }
                            Button("Stop") {
                                Task {
                                    await runServiceAction {
                                        try await state.serviceClient.stopService(id: service.id)
                                    }
                                }
                            }
                            .disabled(!ServiceMenuFormatter.canStopService(
                                pid: service.pid,
                                desiredRunning: service.desiredRunning
                            ))

                            if ServiceMenuFormatter.showEnableAction(enabled: service.enabled) {
                                Button("Enable") {
                                    Task { await runToggleEnable(service: service, enable: true) }
                                }
                            } else {
                                Button("Disable") {
                                    Task { await runToggleEnable(service: service, enable: false) }
                                }
                            }
                        }
                    }
                }
            }
            .accessibilityIdentifier("services-menu")

            Menu("Cron") {
                if !state.hasEndpoint {
                    Text(CronMenuFormatter.formatCronNotConfiguredLabel())
                } else if state.cronTasks.isEmpty {
                    Text(CronMenuFormatter.formatCronTasksEmptyLabel())
                } else {
                    ForEach(state.cronTasks) { task in
                        Menu(CronMenuFormatter.formatCronTaskTitle(
                            name: task.name,
                            status: task.status,
                            enabled: task.enabled,
                            scheduleMode: task.scheduleMode,
                            interval: task.interval,
                            cronExpr: task.cronExpr
                        )) {
                            Button("Run Now") {
                                Task {
                                    await runCronAction {
                                        try await state.serviceClient.runCronTask(id: task.id)
                                    }
                                }
                            }
                            .disabled(!CronMenuFormatter.canRunCronTask(status: task.status))

                            if CronMenuFormatter.showEnableCronAction(enabled: task.enabled) {
                                Button("Enable") {
                                    Task { await runCronToggleEnable(task: task, enable: true) }
                                }
                            } else {
                                Button("Disable") {
                                    Task { await runCronToggleEnable(task: task, enable: false) }
                                }
                            }

                            Button("View Logs…") {
                                // SSE via configured baseURL + Bearer (/api/logs/stream), not keep-alive 23312
                                LogStreamWindow.open(
                                    logPath: task.logPath,
                                    stream: state.serviceClient.streamLog(path: task.logPath, lines: 1000)
                                )
                            }

                            Button("History…") {}
                                .disabled(true)
                        }
                    }
                }
            }
            .accessibilityIdentifier("cron-menu")

            Menu("Terminals") {
                if !state.hasEndpoint {
                    Text("Not configured")
                } else if state.terminals.isEmpty {
                    Text(TerminalMenuFormatter.formatTerminalsEmptyLabel())
                } else {
                    ForEach(state.terminals) { session in
                        Button(TerminalMenuFormatter.formatTerminalTitle(name: session.name, id: session.id, status: session.status)) {
                            state.openAttachTerminal(sessionID: session.id)
                        }
                    }
                }
                Divider()
                Button("New Terminal…") {
                    state.openNewTerminal()
                }
                .disabled(!state.hasEndpoint)
            }
            .accessibilityIdentifier("terminals-menu")

            // Backup submenu: Status (Enable/Disable) · Backup Now · Recent · Reveal in Finder
            Menu("Backup") {
                Menu(state.backupStatusTitle) {
                    Button("Enable") {
                        Task { await state.setBackupEnabled(true) }
                    }
                    .disabled(!BackupMenuFormatter.backupEnableItemEnabled(state.backupEnabled))
                    Button("Disable") {
                        Task { await state.setBackupEnabled(false) }
                    }
                    .disabled(!BackupMenuFormatter.backupDisableItemEnabled(state.backupEnabled))
                }
                .accessibilityIdentifier("backup-status")

                Button("Backup Now…") {
                    Task { await state.runBackupNow() }
                }
                .disabled(!state.hasEndpoint || state.backupRunning)

                Divider()

                if state.backupRecent.isEmpty {
                    Text(BackupMenuFormatter.formatBackupRecentEmptyLabel())
                } else {
                    ForEach(state.backupRecent.prefix(15)) { entry in
                        Text(BackupMenuFormatter.formatBackupEntry(entry))
                    }
                }

                Divider()

                Button("Reveal in Finder…") {
                    state.revealBackupInFinder()
                }
                .disabled(state.backupServerName.isEmpty)
            }
            .accessibilityIdentifier("backup-menu")
            .disabled(!state.hasEndpoint && state.backupServerName.isEmpty)

            Divider()

            Button(OpenInBrowserLabelFormatter.format(browser: defaultBrowser)) {
                if !state.serverURL.isEmpty,
                   let url = URL(string: state.serverURL) {
                    BrowserOpener.open(
                        url: url,
                        browser: BrowserPreference.fromStored(defaultBrowser)
                    )
                }
            }
            .disabled(!state.hasEndpoint || state.serverURL.isEmpty)

            Toggle("Auto Start", isOn: $autoStart)
                .onChange(of: autoStart) { enabled in
                    do {
                        if enabled {
                            try SMAppService.mainApp.register()
                        } else {
                            try SMAppService.mainApp.unregister()
                        }
                    } catch {
                        autoStart = !enabled
                    }
                }
                .accessibilityIdentifier("auto-start-toggle")

            Button("Refresh") {
                Task { await state.refresh() }
            }

            Divider()

            Button("Settings…") {
                showSettings(openWindow)
            }
            .accessibilityIdentifier("settings-menu-button")

            Button("Quit") {
                NSApp.terminate(nil)
            }
        }
        .padding(8)
        .task {
            await state.refresh()
        }
    }

    private func runServiceAction(_ action: @escaping () async throws -> Void) async {
        do {
            try await action()
            await state.refreshServices()
        } catch {
            // Ignore transient server errors; user can retry from the menu.
        }
    }

    private func runToggleEnable(service: ServiceStatus, enable: Bool) async {
        do {
            let response: ServiceActionResponse
            if enable {
                response = try await state.serviceClient.enableService(id: service.id)
            } else {
                response = try await state.serviceClient.disableService(id: service.id)
            }
            let alert = NSAlert()
            alert.messageText = enable ? "Enable Service" : "Disable Service"
            alert.informativeText = response.displayMessage
            alert.alertStyle = .informational
            alert.addButton(withTitle: "OK")
            alert.runModal()
            await state.refreshServices()
        } catch {
            // Ignore transient server errors.
        }
    }

    private func runCronAction(_ action: @escaping () async throws -> Void) async {
        do {
            try await action()
            await state.refreshCronTasks()
        } catch {
            // Ignore transient server errors; user can retry from the menu.
        }
    }

    private func runCronToggleEnable(task: CronTaskStatus, enable: Bool) async {
        do {
            let response: CronTaskActionResponse
            if enable {
                response = try await state.serviceClient.enableCronTask(id: task.id)
            } else {
                response = try await state.serviceClient.disableCronTask(id: task.id)
            }
            let alert = NSAlert()
            alert.messageText = enable ? "Enable Cron Task" : "Disable Cron Task"
            alert.informativeText = response.displayMessage
            alert.alertStyle = .informational
            alert.addButton(withTitle: "OK")
            alert.runModal()
            await state.refreshCronTasks()
        } catch {
            // Ignore transient server errors.
        }
    }
}
