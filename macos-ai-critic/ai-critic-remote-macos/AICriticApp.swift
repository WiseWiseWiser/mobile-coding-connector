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
    @AppStorage("menuBarDisplayMode") var menuBarDisplayMode = "rotating"

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

            // Keep token for authenticated API calls later; never put it in statusLine.
            if result.resolved {
                if let cfg = try RemoteConfigStore.load(path: path) {
                    let (ep, _) = RemoteConfigStore.resolve(cfg)
                    token = ep.token
                }
            } else {
                token = ""
            }
        } catch {
            statusLine = "Not configured — open Configure… to add a remote server"
            serverURL = ""
            hasEndpoint = false
            token = ""
            menuLabel = "Remote …"
        }
    }
}

class RemoteAppDelegate: NSObject, NSApplicationDelegate {
    weak var state: RemoteAppState?

    func applicationDidFinishLaunching(_ notification: Notification) {
        // Stay a menu-bar agent until a window is opened.
        NSApp.setActivationPolicy(.accessory)
        Task { @MainActor in
            await state?.refresh()
        }
    }

    func applicationShouldTerminateAfterLastWindowClosed(_ sender: NSApplication) -> Bool {
        // Menu-bar app must not quit when Settings window closes.
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
        // Same Settings window role as local app (id/title "settings" / "Settings").
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

            Button("Settings…") {
                showSettings(openWindow)
            }
            .accessibilityIdentifier("settings-menu-button")

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

            Button("Quit") {
                NSApp.terminate(nil)
            }
        }
        .padding(8)
        .task {
            await state.refresh()
        }
    }
}
