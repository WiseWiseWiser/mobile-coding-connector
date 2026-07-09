import SwiftUI
import ServiceManagement
import AppKit
import AICriticMacShared

@MainActor
final class AppState: ObservableObject {
    @Published var menuLabel = "Grok ..."
    @Published var grokUsage: GrokUsageResponse?
    @Published var codexUsage: CodexUsageResponse?
    @Published var services: [ServiceStatus] = []
    @Published var daemonStatus: KeepAliveStatus?
    @Published var statusLine = "Connecting..."
    @Published var rotatingIndex = 0
    @AppStorage("menuBarDisplayMode") var menuBarDisplayMode = "rotating"

    func refresh() async {
        async let grokTask: GrokUsageResponse? = {
            do {
                return try await ServerClient.shared.grokUsage()
            } catch {
                return nil
            }
        }()
        async let codexTask: CodexUsageResponse? = {
            do {
                return try await ServerClient.shared.codexUsage()
            } catch {
                return nil
            }
        }()
        async let servicesTask: [ServiceStatus]? = {
            do {
                return try await ServerClient.shared.listServices()
            } catch {
                return nil
            }
        }()

        grokUsage = await grokTask
        codexUsage = await codexTask
        if let listed = await servicesTask {
            services = listed
        }
        updateMenuLabel()

        do {
            let status = try await DaemonClient.shared.keepAliveStatus()
            daemonStatus = status
            if status.running {
                statusLine = "Server running on :\(status.serverPort) (pid \(status.serverPID))"
            } else {
                statusLine = "Keep-alive up; server starting..."
            }
        } catch {
            daemonStatus = nil
            statusLine = "Daemon unreachable on :23312"
        }
    }

    func updateMenuLabel() {
        menuLabel = UsageLabelFormatter.formatMenuBarLabel(
            mode: menuBarDisplayMode,
            rotatingIndex: rotatingIndex,
            grokStatus: grokUsage?.status ?? "loading",
            grokWeekly: grokUsage?.weeklyLimit ?? "",
            grokError: grokUsage?.error ?? "",
            codexStatus: codexUsage?.status ?? "loading",
            codexMonthly: codexUsage?.monthlyUsage ?? "",
            codexError: codexUsage?.error ?? ""
        )
    }

    func advanceRotation() {
        rotatingIndex = (rotatingIndex + 1) % 2
        updateMenuLabel()
    }

    func refreshServices() async {
        do {
            services = try await ServerClient.shared.listServices()
        } catch {
            // Keep prior list when server is temporarily unreachable.
        }
    }
}

class AppDelegate: NSObject, NSApplicationDelegate {
    weak var state: AppState?

    func applicationDidFinishLaunching(_ notification: Notification) {
        Task { @MainActor in
            await DaemonManager.shared.ensureRunning()
            await state?.refresh()
            startRefreshLoop()
            startRotationLoop()
        }
    }

    func applicationWillTerminate(_ notification: Notification) {
        Task { @MainActor in
            DaemonManager.shared.terminateIfSpawned()
        }
    }

    @MainActor
    private func startRefreshLoop() {
        Task { @MainActor in
            while !Task.isCancelled {
                try? await Task.sleep(nanoseconds: 15_000_000_000)
                await state?.refresh()
            }
        }
    }

    @MainActor
    private func startRotationLoop() {
        Task { @MainActor in
            while !Task.isCancelled {
                try? await Task.sleep(nanoseconds: 60_000_000_000)
                state?.advanceRotation()
            }
        }
    }
}

@available(macOS 15.0, *)
@main
struct AICriticApp: App {
    @NSApplicationDelegateAdaptor(AppDelegate.self) var appDelegate
    @StateObject private var state = AppState()
    @AppStorage("autoStart") private var autoStart = false

    init() {
        let state = AppState()
        _state = StateObject(wrappedValue: state)
        appDelegate.state = state
        _autoStart.wrappedValue = SMAppService.mainApp.status == .enabled
    }

    var body: some Scene {
        Window("Settings", id: "settings") {
            LocalSettingsRoot(menuBarDisplayMode: $state.menuBarDisplayMode)
                .onChange(of: state.menuBarDisplayMode) { _ in
                    state.updateMenuLabel()
                }
        }
        .windowResizability(.contentSize)
        .defaultLaunchBehavior(.suppressed)

        MenuBarExtra {
            MenuBarDropdownContent(
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
private struct MenuBarDropdownContent: View {
    @ObservedObject var state: AppState
    @Binding var autoStart: Bool
    @AppStorage("defaultBrowser") private var defaultBrowser = BrowserPreference.default.rawValue
    @Environment(\.openWindow) private var openWindow
    let showSettings: (OpenWindowAction) -> Void

    var body: some View {
        VStack(alignment: .leading, spacing: 6) {
            Text(UsageLabelFormatter.formatGrokDropdownLine(
                status: state.grokUsage?.status ?? "loading",
                weekly: state.grokUsage?.weeklyLimit ?? "",
                reset: state.grokUsage?.nextReset ?? "",
                errorMsg: state.grokUsage?.error ?? "",
                now: Date()
            ))
            Text(UsageLabelFormatter.formatCodexDropdownLine(
                status: state.codexUsage?.status ?? "loading",
                monthly: state.codexUsage?.monthlyUsage ?? "",
                creditsUsed: state.codexUsage?.creditsUsed ?? "",
                creditsTotal: state.codexUsage?.creditsTotal ?? "",
                reset: state.codexUsage?.nextReset ?? "",
                errorMsg: state.codexUsage?.error ?? "",
                now: Date()
            ))

            Text(state.statusLine)
                .font(.caption)
                .foregroundStyle(.secondary)

            Divider()

            Button("Restart Daemon") {
                Task {
                    try? await DaemonClient.shared.restartDaemon()
                    for _ in 0..<10 {
                        await state.refresh()
                        if state.daemonStatus?.running == true {
                            break
                        }
                        try? await Task.sleep(nanoseconds: 500_000_000)
                    }
                }
            }

            Menu("Services") {
                if state.services.isEmpty {
                    Text(ServiceMenuFormatter.formatServicesEmptyLabel())
                } else {
                    ForEach(state.services) { service in
                        Menu(ServiceMenuFormatter.formatServiceTitle(
                            name: service.name,
                            status: service.status,
                            enabled: service.enabled
                        )) {
                            Button("Start") {
                                Task { await runServiceAction(service.id) { try await ServerClient.shared.startService(id: service.id) } }
                            }
                            Button("Restart") {
                                Task { await runServiceAction(service.id) { try await ServerClient.shared.restartService(id: service.id) } }
                            }
                            Button("Stop") {
                                Task { await runServiceAction(service.id) { try await ServerClient.shared.stopService(id: service.id) } }
                            }
                            .disabled(!ServiceMenuFormatter.canStopService(pid: service.pid, desiredRunning: service.desiredRunning))

                            if ServiceMenuFormatter.showEnableAction(enabled: service.enabled) {
                                Button("Enable") {
                                    Task { await runToggleEnable(service: service, enable: true) }
                                }
                            } else {
                                Button("Disable") {
                                    Task { await runToggleEnable(service: service, enable: false) }
                                }
                            }

                            Button("View Logs…") {
                                LogTailWindow.open(logPath: service.logPath)
                            }
                        }
                    }
                }
            }

            Divider()

            Button(OpenInBrowserLabelFormatter.format(browser: defaultBrowser)) {
                if let port = state.daemonStatus?.serverPort,
                   let url = URL(string: "http://127.0.0.1:\(port)") {
                    BrowserOpener.open(
                        url: url,
                        browser: BrowserPreference.fromStored(defaultBrowser)
                    )
                }
            }
            .disabled(state.daemonStatus?.serverPort == nil)

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

            Divider()

            Button("Settings…") {
                showSettings(openWindow)
            }
            .accessibilityIdentifier("settings-menu-button")

            Button("Quit") {
                Task { @MainActor in
                    DaemonManager.shared.terminateIfSpawned()
                    NSApp.terminate(nil)
                }
            }
        }
        .padding(8)
        .task {
            await state.refresh()
        }
    }

    private func runServiceAction(_ id: String, action: @escaping () async throws -> Void) async {
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
                response = try await ServerClient.shared.enableService(id: service.id)
            } else {
                response = try await ServerClient.shared.disableService(id: service.id)
            }
            let alert = NSAlert()
            alert.messageText = enable ? "Enable Service" : "Disable Service"
            alert.informativeText = response.message
            alert.alertStyle = .informational
            alert.addButton(withTitle: "OK")
            alert.runModal()
            await state.refreshServices()
        } catch {
            // Ignore transient server errors.
        }
    }
}