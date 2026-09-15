import SwiftUI

/// Shared Settings window content for local and remote menu-bar apps.
///
/// Common sections: Default Browser.
/// Optional remote connection section (server/token → remote-agent-config.json).
/// Optional extra sections (e.g. local debug log) via `extraSections`.
///
/// The menu-bar usage items are managed by the local-agent `usage` CLI and
/// stored on the server, so there is no display-mode picker here.
public struct SettingsView<Extra: View>: View {
    @AppStorage("defaultBrowser") private var defaultBrowser = BrowserPreference.default.rawValue
    private var showRemoteConnection: Bool
    private var onConnectionSaved: (() -> Void)?
    private var extraSections: Extra

    public init(
        showRemoteConnection: Bool = false,
        onConnectionSaved: (() -> Void)? = nil,
        @ViewBuilder extraSections: () -> Extra
    ) {
        self.showRemoteConnection = showRemoteConnection
        self.onConnectionSaved = onConnectionSaved
        self.extraSections = extraSections()
    }

    public var body: some View {
        VStack(alignment: .leading, spacing: 12) {
            Text("Settings")
                .font(.title2)
                .fontWeight(.semibold)

            Divider()

            if showRemoteConnection {
                ConnectionSettingsSection(onSaved: onConnectionSaved)
                Divider()
            }

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

            extraSections
        }
        .padding(16)
        .frame(maxWidth: .infinity, maxHeight: .infinity, alignment: .topLeading)
        .accessibilityElement(children: .contain)
        .accessibilityIdentifier("settings-window")
    }
}

extension SettingsView where Extra == EmptyView {
    public init(
        showRemoteConnection: Bool = false,
        onConnectionSaved: (() -> Void)? = nil
    ) {
        self.init(
            showRemoteConnection: showRemoteConnection,
            onConnectionSaved: onConnectionSaved,
            extraSections: { EmptyView() }
        )
    }
}
