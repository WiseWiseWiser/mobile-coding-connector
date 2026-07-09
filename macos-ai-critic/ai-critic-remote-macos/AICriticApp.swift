import SwiftUI
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
    @AppStorage("menuBarDisplayMode") var menuBarDisplayMode = "rotating"

    /// Shared HTTP client for remote service APIs (base URL + Bearer token).
    let serviceClient = ServiceClient()

    /// Override for tests; empty means use default CLI path.
    var configPathOverride: String?

    func configPath() -> String {
        if let override = configPathOverride, !override.isEmpty {
            return override
        }
        return RemoteConfigStore.defaultConfigPath()
    }

    func refresh() async {
        let path = configPath()
        do {
            let result = try RemoteConfigStore.statusFromFile(path: path)
            statusLine = result.line
            serverURL = result.server
            hasEndpoint = result.resolved
            menuLabel = result.resolved ? "Remote" : "Remote …"

            if result.resolved {
                if let cfg = try RemoteConfigStore.load(path: path) {
                    let (ep, _) = RemoteConfigStore.resolve(cfg)
                    token = ep.token
                    serviceClient.configure(baseURL: ep.server, token: ep.token)
                }
                await refreshServices()
            } else {
                token = ""
                serviceClient.configure(baseURL: "", token: "")
                services = []
            }
        } catch {
            statusLine = "Not configured — open Configure… to add a remote server"
            serverURL = ""
            hasEndpoint = false
            token = ""
            serviceClient.configure(baseURL: "", token: "")
            services = []
            menuLabel = "Remote …"
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
}

class RemoteAppDelegate: NSObject, NSApplicationDelegate {
    weak var state: RemoteAppState?

    func applicationDidFinishLaunching(_ notification: Notification) {
        NSApp.setActivationPolicy(.accessory)
        Task { @MainActor in
            await state?.refresh()
        }
    }

    func applicationShouldTerminateAfterLastWindowClosed(_ sender: NSApplication) -> Bool {
        false
    }
}

@available(macOS 15.0, *)
@main
struct AICriticRemoteApp: App {
    @NSApplicationDelegateAdaptor(RemoteAppDelegate.self) var appDelegate
    @StateObject private var state = RemoteAppState()

    init() {
        let state = RemoteAppState()
        _state = StateObject(wrappedValue: state)
        appDelegate.state = state
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
}
