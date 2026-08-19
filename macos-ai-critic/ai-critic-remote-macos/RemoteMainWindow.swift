import SwiftUI
import AppKit
import AICriticMacShared

@available(macOS 15.0, *)
struct RemoteMainWindow: View {
    @ObservedObject var state: RemoteAppState
    @Binding var menuBarDisplayMode: String
    @AppStorage("defaultBrowser") private var defaultBrowser = BrowserPreference.default.rawValue

    var body: some View {
        MainWindowChrome { page in
            switch page {
            case .home:
                MainHomePage(
                    statusLine: state.statusLine,
                    browserLabel: OpenInBrowserLabelFormatter.format(browser: defaultBrowser),
                    canOpenBrowser: state.hasEndpoint && !state.serverURL.isEmpty,
                    onOpenBrowser: openBrowser
                )
            case .services:
                ServicesPage(
                    services: state.services,
                    notConfigured: !state.hasEndpoint,
                    onStart: { id in Task { await runService { try await state.serviceClient.startService(id: id) } } },
                    onRestart: { id in Task { await runService { try await state.serviceClient.restartService(id: id) } } },
                    onStop: { id in Task { await runService { try await state.serviceClient.stopService(id: id) } } },
                    onEnable: { id in Task { await toggleEnable(id: id, enable: true) } },
                    onDisable: { id in Task { await toggleEnable(id: id, enable: false) } }
                )
            case .projects:
                ProjectsPage(
                    projects: state.projects,
                    loading: state.projectsLoading,
                    loadError: state.projectsLoadError,
                    notConfigured: !state.hasEndpoint
                )
            case .settings:
                ScrollView {
                    SettingsView(
                        menuBarDisplayMode: $menuBarDisplayMode,
                        showRemoteConnection: true,
                        onConnectionSaved: {
                            Task { @MainActor in
                                await state.refresh()
                            }
                        }
                    )
                }
                .navigationTitle("Settings")
            }
        }
        .modifier(RegisterMainWindowOpener())
    }

    private func openBrowser() {
        if !state.serverURL.isEmpty, let url = URL(string: state.serverURL) {
            BrowserOpener.open(url: url, browser: BrowserPreference.fromStored(defaultBrowser))
        }
    }

    private func runService(_ action: @escaping () async throws -> Void) async {
        try? await action()
        await state.refreshServices()
    }

    private func toggleEnable(id: String, enable: Bool) async {
        do {
            let response: ServiceActionResponse
            if enable {
                response = try await state.serviceClient.enableService(id: id)
            } else {
                response = try await state.serviceClient.disableService(id: id)
            }
            let alert = NSAlert()
            alert.messageText = enable ? "Enable Service" : "Disable Service"
            alert.informativeText = response.displayMessage
            alert.alertStyle = .informational
            alert.addButton(withTitle: "OK")
            alert.runModal()
            await state.refreshServices()
        } catch {
        }
    }
}
