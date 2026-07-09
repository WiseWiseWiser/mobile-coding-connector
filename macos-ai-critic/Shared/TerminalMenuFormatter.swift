import Foundation

/// Terminals menu labels/commands — mirrors `macosapp/menubar` terminal helpers.
public enum TerminalMenuFormatter {
    /// Non-empty trimmed name wins; empty/whitespace name falls back to id.
    public static func formatTerminalTitle(name: String, id: String) -> String {
        let trimmed = name.trimmingCharacters(in: .whitespacesAndNewlines)
        if !trimmed.isEmpty {
            return name
        }
        return id
    }

    public static func formatTerminalsEmptyLabel() -> String {
        "No terminal sessions"
    }

    public static func buildTerminalAttachCommand(agentBinary: String, sessionID: String) -> String {
        "\(agentBinary) terminal attach \(sessionID)"
    }

    public static func buildTerminalNewCommand(agentBinary: String) -> String {
        "\(agentBinary) terminal new"
    }

    public static func agentBinaryForApp(isRemote: Bool) -> String {
        isRemote ? "remote-agent" : "local-agent"
    }

    /// App-side poll period for services + terminals (mirrors Go PeriodicRefreshInterval).
    public static let periodicRefreshIntervalNanoseconds: UInt64 = 30_000_000_000
}
