import SwiftUI

enum MenuBarDisplayMode: String, CaseIterable, Identifiable {
    case rotating
    case grok
    case codex

    var id: String { rawValue }

    var displayName: String {
        switch self {
        case .rotating:
            return "Rotating"
        case .grok:
            return "Grok"
        case .codex:
            return "Codex"
        }
    }
}

struct SettingsView: View {
    @AppStorage("defaultBrowser") private var defaultBrowser = BrowserPreference.default.rawValue
    @AppStorage("debugLogEnabled") private var debugLogEnabled = false
    @Binding var menuBarDisplayMode: String
    @State private var debugLogPath = "/tmp/debug-ai-critic.log"
    @State private var debugSyncError: String?

    var body: some View {
        VStack(alignment: .leading, spacing: 12) {
            Text("Settings")
                .font(.title2)
                .fontWeight(.semibold)

            Divider()

            VStack(alignment: .leading, spacing: 8) {
                Text("Menu Bar Display")
                    .font(.headline)

                Text("Choose which usage appears in the menu bar title:")
                    .font(.caption)
                    .foregroundStyle(.secondary)
                    .fixedSize(horizontal: false, vertical: true)

                Picker("Menu bar display", selection: $menuBarDisplayMode) {
                    ForEach(MenuBarDisplayMode.allCases) { mode in
                        Text(mode.displayName).tag(mode.rawValue)
                    }
                }
                .pickerStyle(.radioGroup)
                .accessibilityIdentifier("menu-bar-display-picker")
            }
            .accessibilityIdentifier("menu-bar-display-section")

            Divider()

            VStack(alignment: .leading, spacing: 8) {
                Text("Default Browser")
                    .font(.headline)

                Text("Choose which browser opens when you click Open in Browser:")
                    .font(.caption)
                    .foregroundStyle(.secondary)
                    .fixedSize(horizontal: false, vertical: true)

                Picker("Open with", selection: $defaultBrowser) {
                    ForEach(BrowserPreference.allCases) { preference in
                        Text(preference.displayName).tag(preference.rawValue)
                    }
                }
                .pickerStyle(.radioGroup)
                .accessibilityIdentifier("browser-picker")
            }
            .accessibilityIdentifier("default-browser-section")

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
        .padding(16)
        .frame(minWidth: 400, minHeight: 340)
        .task {
            await loadDebugLogSettings()
        }
        .accessibilityElement(children: .contain)
        .accessibilityIdentifier("settings-window")
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