import AppKit
import Foundation

/// Append-only monospaced progress/log window for Backup Now (LogStreamWindow family).
/// Quiet open (no focus steal); batched low-CPU UI appends via pending buffer + interval flush.
/// Closing the window does not cancel the in-flight backup job (v1).
public enum BackupProgressWindow {
    private static var sessions: [ObjectIdentifier: ProgressSession] = [:]

    /// Canonical flush interval in milliseconds (100–200ms band; matches menubar helper).
    public static let flushIntervalMilliseconds: Int = 150
    /// Timer interval used by ProgressSession (~0.15s).
    public static let flushInterval: TimeInterval = 0.15

    /// Open a new progress window without stealing keyboard focus from other apps.
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

        // Quiet presentation: show without activating app or stealing key focus.
        window.orderFrontRegardless()
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

        /// Thread-safe pending line buffer; drained on flush.
        private let pendingLock = NSLock()
        private var pendingLines: [String] = []
        private var batchTimer: Timer?
        private let flushInterval: TimeInterval = 0.15

        init(window: NSWindow, textView: NSTextView) {
            self.window = window
            self.textView = textView
            super.init()
            window.delegate = self
        }

        /// Enqueue a line for batched UI append (safe off the main thread).
        public func append(_ line: String) {
            pendingLock.lock()
            pendingLines.append(line)
            pendingLock.unlock()
            // Schedule batch timer on the main run loop if needed.
            if Thread.isMainThread {
                ensureBatchTimer()
            } else {
                DispatchQueue.main.async { [weak self] in
                    self?.ensureBatchTimer()
                }
            }
        }

        public func appendError(_ message: String) {
            append(BackupMenuFormatter.formatBackupProgressError(message: message))
        }

        /// Start or reuse the ~150ms repeating batch timer (main thread only).
        private func ensureBatchTimer() {
            if batchTimer != nil { return }
            let timer = Timer(timeInterval: flushInterval, repeats: true) { [weak self] _ in
                self?.flushPending()
            }
            RunLoop.main.add(timer, forMode: .common)
            batchTimer = timer
        }

        /// Drain pendingLines once via textStorage.append and scroll once.
        private func flushPending() {
            pendingLock.lock()
            let batch = pendingLines
            pendingLines.removeAll(keepingCapacity: true)
            pendingLock.unlock()
            if batch.isEmpty {
                batchTimer?.invalidate()
                batchTimer = nil
                return
            }
            guard let textView else { return }
            let joined = batch.joined(separator: "\n") + "\n"
            let font = textView.font ?? NSFont.monospacedSystemFont(ofSize: 11, weight: .regular)
            textView.textStorage?.append(NSAttributedString(string: joined, attributes: [
                .font: font,
                .foregroundColor: textView.textColor ?? NSColor.textColor,
            ]))
            textView.scrollToEndOfDocument(nil)
        }

        /// Optional interactive bring-to-front (not used on quiet open).
        public func makeKeyAndOrderFront() {
            window?.makeKeyAndOrderFront(nil)
        }

        public func windowWillClose(_ notification: Notification) {
            // Final flush of any remaining pending lines, then tear down timer.
            flushPending()
            batchTimer?.invalidate()
            batchTimer = nil
            // v1: close does not cancel the backup job.
            onClose?()
            onClose = nil
            textView = nil
            window = nil
        }
    }
}
