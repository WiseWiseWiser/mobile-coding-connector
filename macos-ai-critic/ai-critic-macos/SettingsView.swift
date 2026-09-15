import SwiftUI
import AICriticMacShared

/// Local Settings: shared browser section, plus debug log.
/// Thin wrapper around Shared.SettingsView so local keeps ServerClient debug wiring.
@available(macOS 15.0, *)
struct LocalSettingsRoot: View {
    var body: some View {
        SettingsView(showRemoteConnection: false) {
            Divider()
            ITermSwitcherHotKeySection()
            Divider()
            LocalDebugLogSection()
        }
    }
}
