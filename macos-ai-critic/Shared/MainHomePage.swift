import SwiftUI

/// Home page: usage / status + Open in Browser.
@available(macOS 15.0, *)
public struct MainHomePage: View {
    public let statusLine: String
    public let grokLine: String?
    public let codexLine: String?
    public let browserLabel: String
    public let canOpenBrowser: Bool
    public let onOpenBrowser: () -> Void

    public init(
        statusLine: String,
        grokLine: String? = nil,
        codexLine: String? = nil,
        browserLabel: String,
        canOpenBrowser: Bool,
        onOpenBrowser: @escaping () -> Void
    ) {
        self.statusLine = statusLine
        self.grokLine = grokLine
        self.codexLine = codexLine
        self.browserLabel = browserLabel
        self.canOpenBrowser = canOpenBrowser
        self.onOpenBrowser = onOpenBrowser
    }

    public var body: some View {
        ScrollView {
            VStack(alignment: .leading, spacing: 12) {
                if let grokLine {
                    Text(grokLine)
                }
                if let codexLine {
                    Text(codexLine)
                }
                Text(statusLine)
                    .font(.caption)
                    .foregroundStyle(.secondary)
                    .fixedSize(horizontal: false, vertical: true)

                Button(browserLabel, action: onOpenBrowser)
                    .disabled(!canOpenBrowser)
                    .accessibilityIdentifier("home-open-in-browser")
            }
            .frame(maxWidth: .infinity, alignment: .leading)
            .padding(16)
        }
        .navigationTitle("Home")
        .accessibilityIdentifier("home-page")
    }
}
