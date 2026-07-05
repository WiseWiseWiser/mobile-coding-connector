import Darwin
import Foundation

@MainActor
final class DaemonManager: ObservableObject {
    static let shared = DaemonManager()

    @Published private(set) var spawnedByApp = false

    private var daemonProcess: Process?
    private let serverPort = 23712

    private init() {}

    func ensureRunning() async {
        if await DaemonClient.shared.isHealthy() {
            spawnedByApp = false
            return
        }
        spawnDaemon()
        for _ in 0..<40 {
            if await DaemonClient.shared.isHealthy() {
                return
            }
            try? await Task.sleep(nanoseconds: 250_000_000)
        }
    }

    func restart() async {
        if spawnedByApp, let proc = daemonProcess, proc.isRunning {
            proc.terminate()
            proc.waitUntilExit()
            daemonProcess = nil
        }
        spawnedByApp = false
        await ensureRunning()
    }

    func terminateIfSpawned() {
        guard spawnedByApp, let proc = daemonProcess, proc.isRunning else {
            return
        }
        proc.terminate()
        let deadline = Date().addingTimeInterval(3)
        while proc.isRunning && Date() < deadline {
            Thread.sleep(forTimeInterval: 0.05)
        }
        if proc.isRunning {
            kill(proc.processIdentifier, SIGKILL)
        }
        daemonProcess = nil
        spawnedByApp = false
    }

    private func spawnDaemon() {
        let binary = resolveAICriticBinary()
        let home = FileManager.default.homeDirectoryForCurrentUser.path
        let process = Process()
        process.executableURL = URL(fileURLWithPath: binary)
        process.arguments = [
            "keep-alive",
            "--kill-existing",
            "--port", String(serverPort),
        ]
        process.currentDirectoryURL = URL(fileURLWithPath: home)
        let binaryDir = Bundle.main.bundleURL
            .appendingPathComponent("Contents/MacOS")
            .path
        var env = ProcessInfo.processInfo.environment
        env["HOME"] = home
        for (key, value) in Self.keepAliveEnv(binaryDir: binaryDir) {
            env[key] = value
        }
        process.environment = env
        do {
            try process.run()
            daemonProcess = process
            spawnedByApp = true
        } catch {
            print("failed to spawn keep-alive: \(error)")
        }
    }

    private static func keepAliveEnv(binaryDir: String) -> [String: String] {
        _ = binaryDir
        return [
            "AI_CRITIC_NO_OPEN_BROWSER": "1",
        ]
    }

    private func resolveAICriticBinary() -> String {
        if let override = ProcessInfo.processInfo.environment["AI_CRITIC_BIN"], !override.isEmpty {
            return override
        }
        let bundled = Bundle.main.bundleURL
            .appendingPathComponent("Contents/MacOS/ai-critic")
            .path
        if FileManager.default.fileExists(atPath: bundled) {
            return bundled
        }
        let candidates = [
            "/usr/local/bin/ai-critic",
            NSHomeDirectory() + "/go/bin/ai-critic",
        ]
        for path in candidates where FileManager.default.fileExists(atPath: path) {
            return path
        }
        return "/usr/local/bin/ai-critic"
    }
}