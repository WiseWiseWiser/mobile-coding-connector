import AppKit
import Foundation

/// Opens a shell command in **iTerm2 only** (ModeForceNew-style).
/// Never falls back to Terminal.app — missing iTerm surfaces an error/alert path.
public enum ITermOpener {
    public static let appPath = "/Applications/iTerm.app"
    public static let appName = "iTerm2"

    public static func isInstalled() -> Bool {
        FileManager.default.fileExists(atPath: appPath)
    }

    /// Escape text embedded in an AppleScript double-quoted string.
    public static func escapeForAppleScript(_ text: String) -> String {
        text
            .replacingOccurrences(of: "\\", with: "\\\\")
            .replacingOccurrences(of: "\"", with: "\\\"")
    }

    /// Build AppleScript that force-opens a new iTerm2 window and runs `command`.
    public static func buildForceNewWindowScript(command: String) -> String {
        let escaped = escapeForAppleScript(command)
        return """
        tell application "iTerm2"
          activate
          set newWindow to (create window with default profile)
          tell current session of newWindow
            write text "\(escaped)"
          end tell
        end tell
        """
    }

    /// Open iTerm2 and run `command` in a new window.
    /// - Returns: nil on success; an error message if iTerm is missing or osascript fails.
    @discardableResult
    public static func openCommand(_ command: String) -> String? {
        guard isInstalled() else {
            return "iTerm2 is not installed at \(appPath). Install it from https://iterm2.com/"
        }
        let script = buildForceNewWindowScript(command: command)
        let process = Process()
        process.executableURL = URL(fileURLWithPath: "/usr/bin/osascript")
        process.arguments = ["-e", script]
        let errPipe = Pipe()
        process.standardError = errPipe
        process.standardOutput = Pipe()
        do {
            try process.run()
            process.waitUntilExit()
            if process.terminationStatus != 0 {
                let errData = errPipe.fileHandleForReading.readDataToEndOfFile()
                let detail = String(data: errData, encoding: .utf8)?.trimmingCharacters(in: .whitespacesAndNewlines) ?? ""
                if detail.isEmpty {
                    return "Failed to open iTerm2 (osascript exit \(process.terminationStatus))"
                }
                return "Failed to open iTerm2: \(detail)"
            }
            return nil
        } catch {
            return "Failed to open iTerm2: \(error.localizedDescription)"
        }
    }

    /// Run openCommand on the main actor and present an NSAlert on failure.
    @MainActor
    public static func openCommandOrAlert(_ command: String) {
        if let message = openCommand(command) {
            let alert = NSAlert()
            alert.messageText = "iTerm2"
            alert.informativeText = message
            alert.alertStyle = .warning
            alert.addButton(withTitle: "OK")
            alert.runModal()
        }
    }
}
