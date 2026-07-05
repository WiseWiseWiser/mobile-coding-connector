import SwiftUI
import ServiceManagement

@MainActor
final class AppState: ObservableObject {
    @Published var menuLabel = "Grok ..."
    @Published var grokUsage: GrokUsageResponse?
    @Published var codexUsage: CodexUsageResponse?
    @Published var daemonStatus: KeepAliveStatus?
    @Published var statusLine = "Connecting..."
    @Published var rotatingIndex = 0
    @AppStorage("menuBarDisplayMode") var menuBarDisplayMode = "rotating"

    func refresh() async {
        async let grokTask: GrokUsageResponse? = {
            do {
                return try await DaemonClient.shared.grokUsage()
            } catch {
                return nil
            }
        }()
        async let codexTask: CodexUsageResponse? = {
            do {
                return try await DaemonClient.shared.codexUsage()
            } catch {
                return nil
            }
        }()

        grokUsage = await grokTask
        codexUsage = await codexTask
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
            SettingsView(menuBarDisplayMode: $state.menuBarDisplayMode)
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

            Button("Restart Server") {
                Task {
                    try? await DaemonClient.shared.restartServer()
                    await state.refresh()
                }
            }

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
}