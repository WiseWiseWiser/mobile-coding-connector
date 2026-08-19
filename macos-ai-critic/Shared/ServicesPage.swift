import SwiftUI

/// Services list with the same actions as the menu-bar Services submenu.
@available(macOS 15.0, *)
public struct ServicesPage: View {
    public let services: [ServiceStatus]
    public let notConfigured: Bool
    public let onStart: (String) -> Void
    public let onRestart: (String) -> Void
    public let onStop: (String) -> Void
    public let onEnable: (String) -> Void
    public let onDisable: (String) -> Void
    public var onViewLogs: ((String) -> Void)?

    public init(
        services: [ServiceStatus],
        notConfigured: Bool = false,
        onStart: @escaping (String) -> Void,
        onRestart: @escaping (String) -> Void,
        onStop: @escaping (String) -> Void,
        onEnable: @escaping (String) -> Void,
        onDisable: @escaping (String) -> Void,
        onViewLogs: ((String) -> Void)? = nil
    ) {
        self.services = services
        self.notConfigured = notConfigured
        self.onStart = onStart
        self.onRestart = onRestart
        self.onStop = onStop
        self.onEnable = onEnable
        self.onDisable = onDisable
        self.onViewLogs = onViewLogs
    }

    public var body: some View {
        Group {
            if notConfigured {
                ContentUnavailableView("Not configured", systemImage: "server.rack")
            } else if services.isEmpty {
                ContentUnavailableView(
                    ServiceMenuFormatter.formatServicesEmptyLabel(),
                    systemImage: "server.rack"
                )
            } else {
                List(services) { service in
                    VStack(alignment: .leading, spacing: 8) {
                        Text(ServiceMenuFormatter.formatServiceTitle(
                            name: service.name,
                            status: service.status,
                            enabled: service.enabled
                        ))
                        .font(.headline)
                        HStack {
                            Button("Start") { onStart(service.id) }
                            Button("Restart") { onRestart(service.id) }
                            Button("Stop") { onStop(service.id) }
                                .disabled(!ServiceMenuFormatter.canStopService(
                                    pid: service.pid,
                                    desiredRunning: service.desiredRunning
                                ))
                            if ServiceMenuFormatter.showEnableAction(enabled: service.enabled) {
                                Button("Enable") { onEnable(service.id) }
                            } else {
                                Button("Disable") { onDisable(service.id) }
                            }
                            if let onViewLogs {
                                Button("View Logs…") { onViewLogs(service.logPath) }
                            }
                        }
                        .controlSize(.small)
                    }
                    .padding(.vertical, 4)
                }
            }
        }
        .navigationTitle("Services")
        .accessibilityIdentifier("services-page")
    }
}
