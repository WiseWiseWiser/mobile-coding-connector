import SwiftUI
import AICriticMacShared

/// Local Settings: shared browser + Grok/Codex display, plus debug log.
/// Thin wrapper around Shared.SettingsView so local keeps ServerClient debug wiring.
struct LocalSettingsRoot: View {
    @Binding var menuBarDisplayMode: String

    var body: some View {
        SettingsView(
            menuBarDisplayMode: $menuBarDisplayMode,
            showRemoteConnection: false
        ) {
            LocalDebugLogSection()
        }
    }
}
