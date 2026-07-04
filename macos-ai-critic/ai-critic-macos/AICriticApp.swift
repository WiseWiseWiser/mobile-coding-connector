import SwiftUI
import ServiceManagement

@MainActor
final class AppState: ObservableObject {
    @Published var menuLabel = "Grok ..."
    @Published var grokUsage: GrokUsageResponse?
    @Published var daemonStatus: KeepAliveStatus?
    @Published var statusLine = "Connecting..."

    func refresh() async {
        do {
            let usage = try await DaemonClient.shared.grokUsage()
            grokUsage = usage
            menuLabel = GrokLabelFormatter.format(
                status: usage.status,
                weeklyLimit: usage.weeklyLimit ?? "",
                errorMsg: usage.error ?? ""
            )
        } catch {
            menuLabel = "Grok ..."
            grokUsage = nil
        }

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
}

class AppDelegate: NSObject, NSApplicationDelegate {
    weak var state: AppState?

    func applicationDidFinishLaunching(_ notification: Notification) {
        Task { @MainActor in
            await DaemonManager.shared.ensureRunning()
            await state?.refresh()
            startRefreshLoop()
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
        MenuBarExtra {
            MenuBarDropdownContent(state: state, autoStart: $autoStart)
        } label: {
            Text(state.menuLabel)
        }
    }
}

@available(macOS 15.0, *)
private struct MenuBarDropdownContent: View {
    @ObservedObject var state: AppState
    @Binding var autoStart: Bool

    var body: some View {
        VStack(alignment: .leading, spacing: 6) {
            if let usage = state.grokUsage, usage.status == "ready" {
                Text("Weekly limit: \(usage.weeklyLimit ?? "-")")
                Text("Next reset: \(usage.nextReset ?? "-")")
            } else if let usage = state.grokUsage, usage.status == "error" {
                Text("Error: \(usage.error ?? "unknown")")
                    .foregroundStyle(.red)
            } else {
                Text("Loading grok usage...")
                    .foregroundStyle(.secondary)
            }

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

            Button("Open in Browser") {
                if let port = state.daemonStatus?.serverPort {
                    NSWorkspace.shared.open(URL(string: "http://127.0.0.1:\(port)")!)
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