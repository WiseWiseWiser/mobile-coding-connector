import AppKit
import ApplicationServices
import CoreGraphics

/// There is no public AppKit “switch Space” API.
/// What actually changes the *visible* Desktop:
///   1. Focus a window that already lives on that Space (macOS follows it).
///   2. In-process System Events keystrokes (this app is in Accessibility;
///      `osascript` in the daemon is not — that was -25211).
/// HID CGEvent Control+arrow from this process is ignored (logged: active unchanged).
enum SpaceSwitch {
    private static var beacon: NSWindow?

    static func go(
        spaceID: UInt64,
        fromIndex: Int?,
        toIndex: Int,
        firstSessionID: String?,
        focusSession: @escaping (String) async -> Void,
        done: @escaping () -> Void
    ) {
        let trusted = AXIsProcessTrusted()
        ITermSwitcherDebug.log(
            "SpaceSwitch go spaceID=\(spaceID) from=\(fromIndex.map(String.init) ?? "nil") to=\(toIndex) active=\(CGSSpaceMove.activeSpaceID()) ax=\(trusted) session=\(firstSessionID ?? "nil")"
        )

        // Focusing a live window on that Space is the only path that has
        // actually changed this machine's visible Desktop (same as ⏎).
        if let firstSessionID, !firstSessionID.isEmpty {
            Task { @MainActor in
                ITermSwitcherDebug.log("SpaceSwitch focus first session \(firstSessionID)")
                await focusSession(firstSessionID)
                done()
            }
            return
        }

        if goViaBeacon(spaceID: spaceID) {
            poll(want: spaceID, attempts: 12, interval: 0.08) { ok in
                ITermSwitcherDebug.log("SpaceSwitch beacon ok=\(ok) active=\(CGSSpaceMove.activeSpaceID())")
                dropBeacon()
                if ok {
                    done()
                    return
                }
                goViaSystemEventsThenFocus(
                    spaceID: spaceID,
                    fromIndex: fromIndex,
                    toIndex: toIndex,
                    firstSessionID: firstSessionID,
                    focusSession: focusSession,
                    done: done
                )
            }
            return
        }
        goViaSystemEventsThenFocus(
            spaceID: spaceID,
            fromIndex: fromIndex,
            toIndex: toIndex,
            firstSessionID: firstSessionID,
            focusSession: focusSession,
            done: done
        )
    }

    private static func goViaSystemEventsThenFocus(
        spaceID: UInt64,
        fromIndex: Int?,
        toIndex: Int,
        firstSessionID: String?,
        focusSession: @escaping (String) async -> Void,
        done: @escaping () -> Void
    ) {
        Task { @MainActor in
            await stepBySystemEvents(fromIndex: fromIndex, toIndex: toIndex)
            if CGSSpaceMove.activeSpaceID() == spaceID {
                ITermSwitcherDebug.log("SpaceSwitch system events ok")
                done()
                return
            }
            if let firstSessionID, !firstSessionID.isEmpty {
                ITermSwitcherDebug.log("SpaceSwitch fallback focus session \(firstSessionID)")
                await focusSession(firstSessionID)
                done()
                return
            }
            ITermSwitcherDebug.log("SpaceSwitch failed active=\(CGSSpaceMove.activeSpaceID())")
            done()
        }
    }

    /// Own window assigned to the target Space, then made key so macOS follows it.
    @discardableResult
    private static func goViaBeacon(spaceID: UInt64) -> Bool {
        dropBeacon()
        let win = NSWindow(
            contentRect: NSRect(x: 0, y: 0, width: 8, height: 8),
            styleMask: [.borderless],
            backing: .buffered,
            defer: false
        )
        win.isOpaque = false
        win.backgroundColor = .clear
        win.hasShadow = false
        win.level = .floating
        win.collectionBehavior = [.fullScreenAuxiliary, .ignoresCycle, .transient]
        win.ignoresMouseEvents = true
        win.isReleasedWhenClosed = false
        if let screen = NSScreen.main {
            win.setFrameOrigin(NSPoint(x: screen.visibleFrame.midX, y: screen.visibleFrame.midY))
        }
        win.orderFrontRegardless()
        guard CGSSpaceMove.moveWindow(win, toSpaceID: spaceID) else {
            ITermSwitcherDebug.log("SpaceSwitch beacon move failed")
            win.orderOut(nil)
            return false
        }
        beacon = win
        NSApp.setActivationPolicy(.regular)
        NSApp.activate(ignoringOtherApps: true)
        win.makeKeyAndOrderFront(nil)
        ITermSwitcherDebug.log("SpaceSwitch beacon shown win=\(win.windowNumber)")
        return true
    }

    private static func dropBeacon() {
        beacon?.orderOut(nil)
        beacon = nil
    }

    private static func stepBySystemEvents(fromIndex: Int?, toIndex: Int) async {
        guard let from = fromIndex else {
            ITermSwitcherDebug.log("SpaceSwitch no fromIndex")
            return
        }
        let delta = toIndex - from
        guard delta != 0 else { return }
        let code = delta > 0 ? 124 : 123
        let n = abs(delta)
        ITermSwitcherDebug.log("SpaceSwitch AE arrows delta=\(delta) ax=\(AXIsProcessTrusted())")
        for i in 0..<n {
            let err = runSystemEvents("key code \(code) using control down")
            if let err {
                ITermSwitcherDebug.log("SpaceSwitch AE err step \(i): \(err)")
                return
            }
            try? await Task.sleep(nanoseconds: 180_000_000)
        }
        ITermSwitcherDebug.log("SpaceSwitch AE done active=\(CGSSpaceMove.activeSpaceID())")
    }

    private static func runSystemEvents(_ command: String) -> String? {
        let src = "tell application \"System Events\" to \(command)"
        var err: NSDictionary?
        _ = NSAppleScript(source: src)?.executeAndReturnError(&err)
        if let err {
            return String(describing: err)
        }
        return nil
    }

    private static func poll(want: UInt64, attempts: Int, interval: TimeInterval, done: @escaping (Bool) -> Void) {
        var left = attempts
        func tick() {
            if CGSSpaceMove.activeSpaceID() == want {
                done(true)
                return
            }
            left -= 1
            if left <= 0 {
                done(false)
                return
            }
            DispatchQueue.main.asyncAfter(deadline: .now() + interval, execute: tick)
        }
        DispatchQueue.main.asyncAfter(deadline: .now() + interval, execute: tick)
    }
}
