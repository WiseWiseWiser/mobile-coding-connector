import SwiftUI

/// Remote connection editor (server URL + token) saved to remote-agent-config.json.
public struct ConnectionSettingsSection: View {
    public var onSaved: (() -> Void)?

    @State private var server: String = ""
    @State private var token: String = ""
    @State private var statusMessage: String = ""
    @State private var isError: Bool = false
    @State private var loadedConfig: RemoteAgentConfig = RemoteAgentConfig()
    @State private var configPath: String = RemoteConfigStore.defaultConfigPath()

    public init(onSaved: (() -> Void)? = nil) {
        self.onSaved = onSaved
    }

    public var body: some View {
        VStack(alignment: .leading, spacing: 8) {
            Text("Remote Connection")
                .font(.headline)
                .accessibilityIdentifier("remote-connection-section-title")

            Text("Config file: \(configPath)")
                .font(.caption)
                .foregroundStyle(.secondary)
                .textSelection(.enabled)
                .accessibilityIdentifier("remote-config-path")

            Text("Server URL")
                .font(.subheadline)
            TextField("https://host.example.com", text: $server)
                .textFieldStyle(.roundedBorder)
                .accessibilityIdentifier("configure-server-field")

            Text("Token")
                .font(.subheadline)
            SecureField("Bearer token", text: $token)
                .textFieldStyle(.roundedBorder)
                .accessibilityIdentifier("configure-token-field")

            if !statusMessage.isEmpty {
                Text(statusMessage)
                    .font(.caption)
                    .foregroundStyle(isError ? .red : .secondary)
                    .fixedSize(horizontal: false, vertical: true)
            }

            HStack {
                Button("Reload from Disk") {
                    loadFromDisk()
                }
                Spacer()
                Button("Save") {
                    saveToDisk()
                }
                .keyboardShortcut(.defaultAction)
                .accessibilityIdentifier("configure-save-button")
            }
        }
        .accessibilityIdentifier("remote-connection-section")
        .onAppear {
            loadFromDisk()
        }
    }

    private func loadFromDisk() {
        configPath = RemoteConfigStore.defaultConfigPath()
        do {
            if let cfg = try RemoteConfigStore.load(path: configPath) {
                loadedConfig = cfg
                let (ep, _) = RemoteConfigStore.resolve(cfg)
                if ep.ok {
                    server = ep.server
                    token = ep.token
                } else if let first = cfg.domains.first {
                    server = first.server
                    token = first.token
                } else {
                    server = cfg.defaultServer
                    token = ""
                }
                statusMessage = "Loaded \(cfg.domains.count) domain(s)."
                isError = false
            } else {
                loadedConfig = RemoteAgentConfig()
                server = ""
                token = ""
                statusMessage = "No config file yet — enter server and token, then Save."
                isError = false
            }
        } catch {
            statusMessage = "Failed to load: \(error.localizedDescription)"
            isError = true
        }
    }

    private func saveToDisk() {
        let normalized = RemoteConfigStore.normalizeServer(server)
        guard !normalized.isEmpty else {
            statusMessage = "Server URL is required."
            isError = true
            return
        }

        var cfg = loadedConfig
        let tokenValue = token
        var found = false
        for i in cfg.domains.indices {
            if RemoteConfigStore.normalizeServer(cfg.domains[i].server) == normalized {
                cfg.domains[i].server = normalized
                cfg.domains[i].token = tokenValue
                found = true
                break
            }
        }
        if !found {
            if cfg.domains.isEmpty {
                cfg.domains = [RemoteDomain(server: normalized, token: tokenValue)]
            } else if cfg.domains.count == 1 {
                cfg.domains[0] = RemoteDomain(server: normalized, token: tokenValue)
            } else {
                cfg.domains.append(RemoteDomain(server: normalized, token: tokenValue))
            }
        }
        cfg.defaultServer = normalized

        do {
            try RemoteConfigStore.save(path: configPath, config: cfg)
            loadedConfig = cfg
            statusMessage = "Saved."
            isError = false
            onSaved?()
        } catch {
            statusMessage = "Save failed: \(error.localizedDescription)"
            isError = true
        }
    }
}
