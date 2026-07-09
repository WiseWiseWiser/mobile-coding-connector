import SwiftUI
import AICriticMacShared

/// Local-only debug log controls (requires ServerClient on the local daemon).
struct LocalDebugLogSection: View {
    @AppStorage("debugLogEnabled") private var debugLogEnabled = false
    @State private var debugLogPath = "/tmp/debug-ai-critic.log"
    @State private var debugSyncError: String?

    var body: some View {
        Group {
            Divider()

            VStack(alignment: .leading, spacing: 8) {
                Text("Debugging")
                    .font(.headline)

                Toggle("Enable Debugging Log", isOn: $debugLogEnabled)
                    .accessibilityIdentifier("debug-log-toggle")
                    .onChange(of: debugLogEnabled) { enabled in
                        Task { await syncDebugLogSetting(enabled: enabled) }
                    }

                Text("Log file: \(debugLogPath)")
                    .font(.caption)
                    .foregroundStyle(.secondary)
                    .textSelection(.enabled)
                    .accessibilityIdentifier("debug-log-path")

                if let debugSyncError {
                    Text(debugSyncError)
                        .font(.caption)
                        .foregroundStyle(.red)
                }
            }
            .accessibilityIdentifier("debug-log-section")
        }
        .task {
            await loadDebugLogSettings()
        }
    }

    private func loadDebugLogSettings() async {
        do {
            let settings = try await ServerClient.shared.debugLogSettings()
            debugLogEnabled = settings.enabled
            debugLogPath = settings.path
            debugSyncError = nil
        } catch {
            debugSyncError = "Server unreachable; toggle may not apply until the main server is running."
        }
    }

    private func syncDebugLogSetting(enabled: Bool) async {
        do {
            let settings = try await ServerClient.shared.setDebugLog(enabled: enabled)
            debugLogPath = settings.path
            debugSyncError = nil
        } catch {
            debugSyncError = "Failed to update debug log setting on server."
        }
    }
}
