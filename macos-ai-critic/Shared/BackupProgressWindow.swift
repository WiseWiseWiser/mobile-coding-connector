import AppKit
import Foundation

/// Append-only monospaced progress/log window for Backup Now (LogStreamWindow family).
/// Closing the window does not cancel the in-flight backup job (v1).
public enum BackupProgressWindow {
    private static var sessions: [ObjectIdentifier: ProgressSession] = [:]

    /// Open a new progress window (or return an existing controller for append).
    @discardableResult
    public static func open(serverName: String) -> ProgressSession {
        let title = BackupMenuFormatter.formatBackupProgressWindowTitle(serverName: serverName)
        let window = NSWindow(
            contentRect: NSRect(x: 220, y: 180, width: 720, height: 420),
            styleMask: [.titled, .closable, .resizable, .miniaturizable],
            backing: .buffered,
            defer: false
        )
        window.title = title
        window.isReleasedWhenClosed = false

        let scroll = NSScrollView(frame: window.contentView?.bounds ?? .zero)
        scroll.hasVerticalScroller = true
        scroll.hasHorizontalScroller = false
        scroll.autohidesScrollers = true
        scroll.borderType = .noBorder
        scroll.autoresizingMask = [.width, .height]

        let contentSize = scroll.contentSize
        let textView = NSTextView(frame: NSRect(x: 0, y: 0, width: contentSize.width, height: contentSize.height))
        textView.isEditable = false
        textView.isSelectable = true
        textView.font = NSFont.monospacedSystemFont(ofSize: 11, weight: .regular)
        textView.autoresizingMask = [.width]
        textView.isVerticallyResizable = true
        textView.isHorizontallyResizable = false
        textView.textContainer?.containerSize = NSSize(width: contentSize.width, height: CGFloat.greatestFiniteMagnitude)
        textView.textContainer?.widthTracksTextView = true
        scroll.documentView = textView
        window.contentView = scroll

        let session = ProgressSession(window: window, textView: textView)
        sessions[ObjectIdentifier(window)] = session
        session.onClose = {
            sessions.removeValue(forKey: ObjectIdentifier(window))
        }

        window.makeKeyAndOrderFront(nil)
        NSApp.activate(ignoringOtherApps: true)
        return session
    }

    /// Convenience: open and append the standard start header lines.
    @discardableResult
    public static func openBackupProgress(serverName: String, startedAt: Date = Date()) -> ProgressSession {
        let session = open(serverName: serverName)
        session.append(BackupMenuFormatter.formatBackupProgressStartHeader(serverName: serverName))
        session.append(BackupMenuFormatter.formatBackupProgressStartedAt(startedAt))
        return session
    }

    public final class ProgressSession: NSObject, NSWindowDelegate {
        private weak var window: NSWindow?
        private weak var textView: NSTextView?
        var onClose: (() -> Void)?

        init(window: NSWindow, textView: NSTextView) {
            self.window = window
            self.textView = textView
            super.init()
            window.delegate = self
        }

        public func append(_ line: String) {
            if Thread.isMainThread {
                appendOnMain(line)
            } else {
                DispatchQueue.main.async { [weak self] in
                    self?.appendOnMain(line)
                }
            }
        }

        public func appendError(_ message: String) {
            append(BackupMenuFormatter.formatBackupProgressError(message: message))
        }

        private func appendOnMain(_ line: String) {
            guard let textView else { return }
            if textView.string.isEmpty {
                textView.string = line + "\n"
            } else {
                textView.string += line + "\n"
            }
            textView.scrollToEndOfDocument(nil)
        }

        public func makeKeyAndOrderFront() {
            window?.makeKeyAndOrderFront(nil)
        }

        public func windowWillClose(_ notification: Notification) {
            // v1: close does not cancel the backup job.
            onClose?()
            onClose = nil
            textView = nil
            window = nil
        }
    }
}
