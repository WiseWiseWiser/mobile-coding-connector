import Foundation

/// Pure config helpers for the remote menu-bar app.
/// Mirrors `macosapp/remoteconfig` and CLI `~/.ai-critic/remote-agent-config.json`.
public struct RemoteDomain: Codable, Equatable, Identifiable {
    public var server: String
    public var token: String

    public var id: String { server }

    enum CodingKeys: String, CodingKey {
        case server
        case token
    }

    public init(server: String = "", token: String = "") {
        self.server = server
        self.token = token
    }

    public init(from decoder: Decoder) throws {
        let c = try decoder.container(keyedBy: CodingKeys.self)
        server = try c.decodeIfPresent(String.self, forKey: .server) ?? ""
        token = try c.decodeIfPresent(String.self, forKey: .token) ?? ""
    }
}

public struct RemoteProjectBinding: Codable, Equatable {
    public var server: String
    public var remoteDir: String
    public var localPath: String

    enum CodingKeys: String, CodingKey {
        case server
        case remoteDir = "remote_dir"
        case localPath = "local_path"
    }

    public init(server: String = "", remoteDir: String = "", localPath: String = "") {
        self.server = server
        self.remoteDir = remoteDir
        self.localPath = localPath
    }
}

public struct RemoteAgentConfig: Codable, Equatable {
    public var defaultServer: String
    public var domains: [RemoteDomain]
    public var projectBindings: [RemoteProjectBinding]

    enum CodingKeys: String, CodingKey {
        case defaultServer = "default"
        case domains
        case projectBindings = "project_bindings"
    }

    public init(
        defaultServer: String = "",
        domains: [RemoteDomain] = [],
        projectBindings: [RemoteProjectBinding] = []
    ) {
        self.defaultServer = defaultServer
        self.domains = domains
        self.projectBindings = projectBindings
    }

    public init(from decoder: Decoder) throws {
        let c = try decoder.container(keyedBy: CodingKeys.self)
        defaultServer = try c.decodeIfPresent(String.self, forKey: .defaultServer) ?? ""
        domains = try c.decodeIfPresent([RemoteDomain].self, forKey: .domains) ?? []
        projectBindings = try c.decodeIfPresent([RemoteProjectBinding].self, forKey: .projectBindings) ?? []
    }

    public func encode(to encoder: Encoder) throws {
        var c = encoder.container(keyedBy: CodingKeys.self)
        if !defaultServer.isEmpty {
            try c.encode(defaultServer, forKey: .defaultServer)
        }
        try c.encode(domains, forKey: .domains)
        if !projectBindings.isEmpty {
            try c.encode(projectBindings, forKey: .projectBindings)
        }
    }
}

public enum RemoteConnectionState: String {
    case notConfigured = "not_configured"
    case noDefault = "no_default"
    case ok = "ok"
    case unauthorized = "unauthorized"
    case unreachable = "unreachable"
}

public struct RemoteResolvedEndpoint {
    public var server: String = ""
    public var token: String = ""
    public var ok: Bool = false

    public init(server: String = "", token: String = "", ok: Bool = false) {
        self.server = server
        self.token = token
        self.ok = ok
    }
}

public enum RemoteConfigStore {
    public static let configFileName = "remote-agent-config.json"

    /// Same path as remote-agent CLI: `$HOME/.ai-critic/remote-agent-config.json`.
    public static func defaultConfigPath(home: String = NSHomeDirectory()) -> String {
        let dir = (home as NSString).appendingPathComponent(".ai-critic")
        return (dir as NSString).appendingPathComponent(configFileName)
    }

    public static func normalizeServer(_ server: String) -> String {
        var s = server.trimmingCharacters(in: .whitespacesAndNewlines)
        while s.hasSuffix("/") {
            s.removeLast()
        }
        return s
    }

    public static func load(path: String) throws -> RemoteAgentConfig? {
        let fm = FileManager.default
        guard fm.fileExists(atPath: path) else {
            return nil
        }
        let data = try Data(contentsOf: URL(fileURLWithPath: path))
        return try JSONDecoder().decode(RemoteAgentConfig.self, from: data)
    }

    public static func save(path: String, config: RemoteAgentConfig) throws {
        let dir = (path as NSString).deletingLastPathComponent
        try FileManager.default.createDirectory(
            atPath: dir,
            withIntermediateDirectories: true,
            attributes: nil
        )
        var out = config
        if out.domains.isEmpty {
            out.domains = []
        }
        let encoder = JSONEncoder()
        encoder.outputFormatting = [.prettyPrinted, .sortedKeys]
        var data = try encoder.encode(out)
        data.append(contentsOf: "\n".utf8)
        try data.write(to: URL(fileURLWithPath: path), options: .atomic)
        try FileManager.default.setAttributes(
            [.posixPermissions: 0o600],
            ofItemAtPath: path
        )
    }

    public static func resolve(_ config: RemoteAgentConfig?) -> (RemoteResolvedEndpoint, RemoteConnectionState) {
        guard let config, !config.domains.isEmpty else {
            return (RemoteResolvedEndpoint(), .notConfigured)
        }

        let def = normalizeServer(config.defaultServer)
        if !def.isEmpty {
            for d in config.domains {
                if normalizeServer(d.server) == def {
                    return (endpoint(from: d), .ok)
                }
            }
        }

        if config.domains.count == 1 {
            return (endpoint(from: config.domains[0]), .ok)
        }

        return (RemoteResolvedEndpoint(), .noDefault)
    }

    private static func endpoint(from d: RemoteDomain) -> RemoteResolvedEndpoint {
        RemoteResolvedEndpoint(
            server: normalizeServer(d.server),
            token: d.token,
            ok: true
        )
    }

    public static func formatStatus(state: RemoteConnectionState, server: String) -> String {
        // Keep copy aligned with macosapp/remoteconfig (Configure… wording).
        switch state {
        case .notConfigured:
            return "Not configured — open Configure… to add a remote server"
        case .noDefault:
            return "Multiple servers configured — open Configure… to pick a default"
        case .ok:
            return "Connected to " + server
        case .unauthorized:
            return "Token rejected — open Configure… to update credentials"
        case .unreachable:
            return "Cannot reach " + server + " — retry or Test Connection"
        }
    }

    /// Shipped refresh path: load file → resolve → status line (mirrors Go StatusFromFile).
    public static func statusFromFile(path: String) throws -> (line: String, server: String, resolved: Bool) {
        let cfg = try load(path: path)
        let (ep, state) = resolve(cfg)
        return (formatStatus(state: state, server: ep.server), ep.server, ep.ok)
    }

    /// Select a domain as default (mirrors Go `remoteconfig.SelectDefaultDomain`).
    /// Sets `default` to the matching domain's normalized server URL.
    public static func selectDefaultDomain(
        _ config: RemoteAgentConfig,
        serverURL: String
    ) throws -> RemoteAgentConfig {
        let norm = normalizeServer(serverURL)
        guard !norm.isEmpty else {
            throw RemoteConfigError.domainNotFound(serverURL)
        }
        guard let match = config.domains.first(where: { normalizeServer($0.server) == norm }) else {
            throw RemoteConfigError.domainNotFound(serverURL)
        }
        var out = config
        out.defaultServer = normalizeServer(match.server)
        return out
    }
}

public enum RemoteConfigError: LocalizedError {
    case domainNotFound(String)

    public var errorDescription: String? {
        switch self {
        case .domainNotFound(let server):
            return "Domain not found: \(server)"
        }
    }
}
