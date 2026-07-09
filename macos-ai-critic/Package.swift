// swift-tools-version: 5.9
import PackageDescription

let package = Package(
    name: "ai-critic-macos",
    platforms: [.macOS(.v13)],
    products: [
        .executable(name: "ai-critic-macos", targets: ["ai-critic-macos"]),
        .executable(name: "ai-critic-remote-macos", targets: ["ai-critic-remote-macos"]),
    ],
    targets: [
        .target(
            name: "AICriticMacShared",
            path: "Shared"
        ),
        .executableTarget(
            name: "ai-critic-macos",
            dependencies: ["AICriticMacShared"],
            path: "ai-critic-macos"
        ),
        .executableTarget(
            name: "ai-critic-remote-macos",
            dependencies: ["AICriticMacShared"],
            path: "ai-critic-remote-macos"
        ),
    ]
)
