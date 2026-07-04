// swift-tools-version: 5.9
import PackageDescription

let package = Package(
    name: "ai-critic-macos",
    platforms: [.macOS(.v13)],
    products: [
        .executable(name: "ai-critic-macos", targets: ["ai-critic-macos"]),
    ],
    targets: [
        .executableTarget(
            name: "ai-critic-macos",
            path: "ai-critic-macos"
        ),
    ]
)