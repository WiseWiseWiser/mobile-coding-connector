import SwiftUI

/// Shared Settings window content for local and remote menu-bar apps.
///
/// Common sections: Menu Bar Display (Grok/Codex/rotating) and Default Browser.
/// Optional remote connection section (server/token → remote-agent-config.json).
/// Optional extra sections (e.g. local debug log) via `extraSections`.
public struct SettingsView<Extra: View>: View {
    @AppStorage("defaultBrowser") private var defaultBrowser = BrowserPreference.default.rawValue
    @Binding private var menuBarDisplayMode: String
    private var showRemoteConnection: Bool
    private var onConnectionSaved: (() -> Void)?
    private var extraSections: Extra

    public init(
        menuBarDisplayMode: Binding<String>,
        showRemoteConnection: Bool = false,
        onConnectionSaved: (() -> Void)? = nil,
        @ViewBuilder extraSections: () -> Extra
    ) {
        self._menuBarDisplayMode = menuBarDisplayMode
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

            extraSections
        }
        .padding(16)
        .frame(minWidth: 440, minHeight: showRemoteConnection ? 420 : 340)
        .accessibilityElement(children: .contain)
        .accessibilityIdentifier("settings-window")
    }
}

extension SettingsView where Extra == EmptyView {
    public init(
        menuBarDisplayMode: Binding<String>,
        showRemoteConnection: Bool = false,
        onConnectionSaved: (() -> Void)? = nil
    ) {
        self.init(
            menuBarDisplayMode: menuBarDisplayMode,
            showRemoteConnection: showRemoteConnection,
            onConnectionSaved: onConnectionSaved,
            extraSections: { EmptyView() }
        )
    }
}
