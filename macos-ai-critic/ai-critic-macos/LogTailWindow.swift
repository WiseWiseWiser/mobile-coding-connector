import AppKit
import Foundation

enum LogTailWindow {
    private static var sessions: [ObjectIdentifier: StreamSession] = [:]

    static func open(logPath: String) {
        let window = NSWindow(
            contentRect: NSRect(x: 200, y: 200, width: 720, height: 420),
            styleMask: [.titled, .closable, .resizable, .miniaturizable],
            backing: .buffered,
            defer: false
        )
        window.title = "Logs: \(URL(fileURLWithPath: logPath).lastPathComponent)"
        window.isReleasedWhenClosed = false

        let textView = NSTextView(frame: window.contentView?.bounds ?? .zero)
        textView.isEditable = false
        textView.font = NSFont.monospacedSystemFont(ofSize: 11, weight: .regular)
        textView.autoresizingMask = [.width, .height]
        window.contentView?.addSubview(textView)

        let session = StreamSession()
        sessions[ObjectIdentifier(window)] = session
        session.start(logPath: logPath, textView: textView, window: window) {
            sessions.removeValue(forKey: ObjectIdentifier(window))
        }

        window.makeKeyAndOrderFront(nil)
    }

    private final class StreamSession: NSObject, NSWindowDelegate {
        private var streamTask: Task<Void, Never>?
        private weak var textView: NSTextView?
        private var onClose: (() -> Void)?

        func start(logPath: String, textView: NSTextView, window: NSWindow, onClose: @escaping () -> Void) {
            self.textView = textView
            self.onClose = onClose
            window.delegate = self

            streamTask = Task {
                do {
                    let stream = ServerClient.shared.streamLog(path: logPath, lines: 1000)
                    for try await event in stream {
                        guard !Task.isCancelled else { break }
                        switch event.type {
                        case "log":
                            if let message = event.message {
                                await appendLine(message)
                            }
                        case "error":
                            if let message = event.message {
                                await appendLine(message, isError: true)
                            }
                        default:
                            break
                        }
                    }
                } catch {
                    guard !Task.isCancelled else { return }
                    await appendLine("Stream error: \(error.localizedDescription)", isError: true)
                }
            }
        }

        @MainActor
        private func appendLine(_ line: String, isError: Bool = false) {
            guard let textView else { return }
            if isError {
                textView.string += "\n\(line)\n"
            } else {
                textView.string += line + "\n"
            }
            textView.scrollToEndOfDocument(nil)
        }

        func windowWillClose(_ notification: Notification) {
            streamTask?.cancel()
            streamTask = nil
            onClose?()
            onClose = nil
        }
    }
}