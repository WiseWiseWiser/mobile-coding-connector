import SwiftUI
import AppKit
import AICriticMacShared

@available(macOS 15.0, *)
struct LocalMainWindow: View {
    @ObservedObject var state: AppState
    @Binding var menuBarDisplayMode: String
    @AppStorage("defaultBrowser") private var defaultBrowser = BrowserPreference.default.rawValue

    var body: some View {
        MainWindowChrome { page in
            switch page {
            case .home:
                MainHomePage(
                    statusLine: state.statusLine,
                    grokLine: UsageLabelFormatter.composeGrokDropdownLine(
                        status: state.grokUsage?.status ?? "loading",
                        weekly: state.grokUsage?.weeklyLimit ?? "",
                        resetDisplay: state.grokUsage?.resetDisplay ?? "",
                        timeLeft: state.grokUsage?.timeLeft ?? "",
                        errorMsg: state.grokUsage?.error ?? ""
                    ),
                    codexLine: UsageLabelFormatter.composeCodexDropdownLine(
                        status: state.codexUsage?.status ?? "loading",
                        monthly: state.codexUsage?.monthlyUsage ?? "",
                        creditsUsed: state.codexUsage?.creditsUsed ?? "",
                        creditsTotal: state.codexUsage?.creditsTotal ?? "",
                        resetDisplay: state.codexUsage?.resetDisplay ?? "",
                        timeLeft: state.codexUsage?.timeLeft ?? "",
                        errorMsg: state.codexUsage?.error ?? ""
                    ),
                    browserLabel: OpenInBrowserLabelFormatter.format(browser: defaultBrowser),
                    canOpenBrowser: state.daemonStatus?.serverPort != nil,
                    onOpenBrowser: openBrowser
                )
            case .services:
                ServicesPage(
                    services: state.services,
                    onStart: { id in Task { await runService { try await ServerClient.shared.startService(id: id) } } },
                    onRestart: { id in Task { await runService { try await ServerClient.shared.restartService(id: id) } } },
                    onStop: { id in Task { await runService { try await ServerClient.shared.stopService(id: id) } } },
                    onEnable: { id in Task { await toggleEnable(id: id, enable: true) } },
                    onDisable: { id in Task { await toggleEnable(id: id, enable: false) } },
                    onViewLogs: { path in LogTailWindow.open(logPath: path) }
                )
            case .projects:
                ProjectsPage(
                    projects: state.projects,
                    loading: state.projectsLoading,
                    loadError: state.projectsLoadError,
                    onOpenInITerm: { path in state.openProjectInITerm2(path: path) },
                    onNewWorktree: { project in
                        Task {
                            guard let taskText = promptNewWorktree() else { return }
                            let task: String? = taskText.isEmpty ? nil : taskText
                            await state.createWorktree(for: project, task: task)
                        }
                    }
                )
            case .settings:
                ScrollView {
                    LocalSettingsRoot(menuBarDisplayMode: $menuBarDisplayMode)
                }
                .navigationTitle("Settings")
            }
        }
        .modifier(RegisterMainWindowOpener())
        .onChange(of: menuBarDisplayMode) { _ in
            state.updateMenuLabel()
        }
    }

    private func openBrowser() {
        if let port = state.daemonStatus?.serverPort,
           let url = URL(string: "http://127.0.0.1:\(port)") {
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
                response = try await ServerClient.shared.enableService(id: id)
            } else {
                response = try await ServerClient.shared.disableService(id: id)
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

    private func promptNewWorktree() -> String? {
        let alert = NSAlert()
        alert.messageText = "New Worktree"
        alert.informativeText = "Optional task description (used as a path/branch slug)."
        alert.alertStyle = .informational
        alert.addButton(withTitle: "Create")
        alert.addButton(withTitle: "Cancel")
        let field = NSTextField(frame: NSRect(x: 0, y: 0, width: 260, height: 24))
        field.placeholderString = "e.g. Fix Login"
        alert.accessoryView = field
        guard alert.runModal() == .alertFirstButtonReturn else { return nil }
        return field.stringValue.trimmingCharacters(in: .whitespacesAndNewlines)
    }
}
