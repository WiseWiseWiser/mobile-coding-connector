import AppKit
import SwiftUI

/// Opens / fronts the main window. Menu-bar extras often stay accessory until
/// activation policy is regular and the window is ordered front.
@MainActor
public enum MainWindowController {
    public static let openNotification = Notification.Name("ai-critic.openMainWindow")
    public static let windowID = "main"

    /// Captured from a SwiftUI scene that has `@Environment(\.openWindow)`.
    public static var openWindowAction: OpenWindowAction?

    public static func registerOpenWindow(_ action: OpenWindowAction) {
        openWindowAction = action
    }

    /// When `page` is nil, keep the last remembered sidebar item.
    public static func open(page: MainSidebarItem? = nil) {
        if let page {
            MainWindowRouter.shared.open(page)
        }
        NSApp.setActivationPolicy(.regular)
        NSApp.activate(ignoringOtherApps: true)

        if let openWindow = openWindowAction {
            openWindow(id: windowID)
        } else {
            NotificationCenter.default.post(name: openNotification, object: nil)
        }

        if let window = findMainWindow() {
            window.makeKeyAndOrderFront(nil)
            window.orderFrontRegardless()
        }
    }

    public static func openWithRetry(attempts: Int = 30, delayNs: UInt64 = 50_000_000) {
        Task { @MainActor in
            for _ in 0..<attempts {
                open()
                if findMainWindow() != nil {
                    return
                }
                try? await Task.sleep(nanoseconds: delayNs)
            }
            open()
        }
    }

    public static func findMainWindow() -> NSWindow? {
        for window in NSApp.windows {
            if let id = window.identifier?.rawValue, id == windowID {
                return window
            }
            if window.title == "AI Critic" || window.title.hasPrefix("AI Critic") {
                return window
            }
            if window.isRestorable,
               window.styleMask.contains(.titled),
               !window.styleMask.contains(.nonactivatingPanel),
               window.contentView != nil,
               window.frame.width >= 400 {
                return window
            }
        }
        return nil
    }
}

/// Captures `openWindow` and reacts to AppDelegate open requests.
@available(macOS 15.0, *)
public struct RegisterMainWindowOpener: ViewModifier {
    @Environment(\.openWindow) private var openWindow

    public init() {}

    public func body(content: Content) -> some View {
        content
            .onAppear {
                MainWindowController.registerOpenWindow(openWindow)
            }
            .onReceive(NotificationCenter.default.publisher(for: MainWindowController.openNotification)) { _ in
                MainWindowController.registerOpenWindow(openWindow)
                openWindow(id: MainWindowController.windowID)
                if let window = MainWindowController.findMainWindow() {
                    window.makeKeyAndOrderFront(nil)
                    window.orderFrontRegardless()
                }
            }
    }
}
