import Foundation

/// Terminals menu labels/commands — mirrors `macosapp/menubar` terminal helpers.
public enum TerminalMenuFormatter {
    /// Non-empty trimmed name wins; empty/whitespace name falls back to id.
    /// When status is "exited" (case-insensitive, trimmed), appends " [EXITED]".
    public static func formatTerminalTitle(name: String, id: String, status: String = "") -> String {
        let trimmed = name.trimmingCharacters(in: .whitespacesAndNewlines)
        let base = trimmed.isEmpty ? id : name
        let statusTrimmed = status.trimmingCharacters(in: .whitespacesAndNewlines)
        if statusTrimmed.caseInsensitiveCompare("exited") == .orderedSame {
            return base + " [EXITED]"
        }
        return base
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
