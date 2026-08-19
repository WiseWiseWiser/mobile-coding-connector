import AppKit
import CoreFoundation
import Darwin

/// Pins an NSWindow onto one Mission Control Space via SkyLight.
enum CGSSpaceMove {
    @discardableResult
    static func moveWindow(_ window: NSWindow, toSpaceID spaceID: UInt64) -> Bool {
        guard spaceID != 0 else { return false }
        let windowID = UInt32(window.windowNumber)
        guard windowID != 0 else { return false }
        return moveWindowID(windowID, toSpaceID: spaceID)
    }

    static func moveWindowID(_ windowID: UInt32, toSpaceID spaceID: UInt64) -> Bool {
        guard let sky = dlopen("/System/Library/PrivateFrameworks/SkyLight.framework/SkyLight", RTLD_LAZY) else {
            return false
        }
        typealias MainConn = @convention(c) () -> Int32
        typealias MoveFn = @convention(c) (Int32, CFArray, UInt64) -> Int32
        let mainSym = dlsym(sky, "CGSMainConnectionID") ?? dlsym(sky, "SLSMainConnectionID")
        let moveSym = dlsym(sky, "CGSMoveWindowsToManagedSpace") ?? dlsym(sky, "SLSMoveWindowsToManagedSpace")
        guard let mainSym, let moveSym else { return false }
        let mainConn = unsafeBitCast(mainSym, to: MainConn.self)
        let move = unsafeBitCast(moveSym, to: MoveFn.self)
        let cid = mainConn()
        let num = windowID as NSNumber
        let arr = [num] as CFArray
        return move(cid, arr, spaceID) == 0
    }

    static func activeSpaceID() -> UInt64 {
        guard let sky = dlopen("/System/Library/PrivateFrameworks/SkyLight.framework/SkyLight", RTLD_LAZY) else {
            return 0
        }
        typealias MainConn = @convention(c) () -> Int32
        typealias GetActive = @convention(c) (Int32) -> UInt64
        let mainSym = dlsym(sky, "CGSMainConnectionID") ?? dlsym(sky, "SLSMainConnectionID")
        let activeSym = dlsym(sky, "CGSGetActiveSpace") ?? dlsym(sky, "SLSGetActiveSpace")
        guard let mainSym, let activeSym else { return 0 }
        let cid = unsafeBitCast(mainSym, to: MainConn.self)()
        return unsafeBitCast(activeSym, to: GetActive.self)(cid)
    }
}
