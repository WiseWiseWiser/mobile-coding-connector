import SwiftUI

/// Home page: usage / status + Open in Browser.
@available(macOS 15.0, *)
public struct MainHomePage: View {
    public let statusLine: String
    /// One line per enabled usage item, printed verbatim from the server.
    public let usageLines: [String]
    public let browserLabel: String
    public let canOpenBrowser: Bool
    public let onOpenBrowser: () -> Void

    public init(
        statusLine: String,
        usageLines: [String] = [],
        browserLabel: String,
        canOpenBrowser: Bool,
        onOpenBrowser: @escaping () -> Void
    ) {
        self.statusLine = statusLine
        self.usageLines = usageLines
        self.browserLabel = browserLabel
        self.canOpenBrowser = canOpenBrowser
        self.onOpenBrowser = onOpenBrowser
    }

    public var body: some View {
        ScrollView {
            VStack(alignment: .leading, spacing: 12) {
                ForEach(Array(usageLines.enumerated()), id: \.offset) { _, line in
                    Text(line)
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
