import Foundation

/// Pure helpers for resolving a Bearer token for the local loopback server.
/// Mirrors `macosapp/localauth`: config first, then server-credentials first line.
enum LocalAuth {
    static let configFileName = "local-agent-config.json"
    static let credentialsFileName = "server-credentials"

    /// Local loopback servers after normalize (trim space + trailing `/`).
    static let localLoopbackServers = [
        "http://localhost:23712",
        "http://127.0.0.1:23712",
    ]

    enum TokenSource: String {
        case config
        case credentials
        case none
    }

    struct Options {
        var dataDir: String = ""
        var configPath: String = ""
        var credentialsPath: String = ""
    }

    private struct ConfigFile: Decodable {
        var defaultServer: String
        var domains: [Domain]

        enum CodingKeys: String, CodingKey {
            case defaultServer = "default"
            case domains
        }

        init(from decoder: Decoder) throws {
            let c = try decoder.container(keyedBy: CodingKeys.self)
            defaultServer = try c.decodeIfPresent(String.self, forKey: .defaultServer) ?? ""
            domains = try c.decodeIfPresent([Domain].self, forKey: .domains) ?? []
        }
    }

    private struct Domain: Decodable {
        var server: String
        var token: String

        enum CodingKeys: String, CodingKey {
            case server, token
        }

        init(from decoder: Decoder) throws {
            let c = try decoder.container(keyedBy: CodingKeys.self)
            server = try c.decodeIfPresent(String.self, forKey: .server) ?? ""
            token = try c.decodeIfPresent(String.self, forKey: .token) ?? ""
        }
    }

    /// Resolve token from `~/.ai-critic` (or opts.dataDir): config → credentials → none.
    static func resolveLocalServerToken(opts: Options = Options()) -> (token: String, source: TokenSource) {
        var dataDir = opts.dataDir
        if dataDir.isEmpty {
            dataDir = (NSHomeDirectory() as NSString).appendingPathComponent(".ai-critic")
        }

        let configPath = opts.configPath.isEmpty
            ? (dataDir as NSString).appendingPathComponent(configFileName)
            : opts.configPath
        let credsPath = opts.credentialsPath.isEmpty
            ? (dataDir as NSString).appendingPathComponent(credentialsFileName)
            : opts.credentialsPath

        if let token = tokenFromConfig(path: configPath) {
            return (token, .config)
        }
        if let token = tokenFromCredentials(path: credsPath) {
            return (token, .credentials)
        }
        return ("", .none)
    }

    /// Formats Authorization header value: `Bearer <token>` or empty when token is empty.
    static func authorizationHeader(token: String) -> String {
        token.isEmpty ? "" : "Bearer \(token)"
    }

    static func normalizeServer(_ server: String) -> String {
        var s = server.trimmingCharacters(in: .whitespacesAndNewlines)
        while s.hasSuffix("/") {
            s.removeLast()
        }
        return s
    }

    private static func tokenFromConfig(path: String) -> String? {
        guard FileManager.default.fileExists(atPath: path),
              let data = try? Data(contentsOf: URL(fileURLWithPath: path)),
              let cfg = try? JSONDecoder().decode(ConfigFile.self, from: data)
        else {
            return nil
        }

        // Prefer local loopback domain with non-empty trimmed token.
        for target in localLoopbackServers {
            if let t = domainTokenMatching(cfg.domains, wantNormalized: target) {
                return t
            }
        }

        // Else default domain.
        let def = normalizeServer(cfg.defaultServer)
        if !def.isEmpty, let t = domainTokenMatching(cfg.domains, wantNormalized: def) {
            return t
        }
        return nil
    }

    private static func domainTokenMatching(_ domains: [Domain], wantNormalized: String) -> String? {
        for d in domains {
            guard normalizeServer(d.server) == wantNormalized else { continue }
            let t = d.token.trimmingCharacters(in: .whitespacesAndNewlines)
            if !t.isEmpty {
                return t
            }
        }
        return nil
    }

    private static func tokenFromCredentials(path: String) -> String? {
        guard FileManager.default.fileExists(atPath: path),
              let content = try? String(contentsOfFile: path, encoding: .utf8)
        else {
            return nil
        }
        for line in content.split(separator: "\n", omittingEmptySubsequences: false) {
            let t = line.trimmingCharacters(in: .whitespacesAndNewlines)
            if !t.isEmpty {
                return t
            }
        }
        return nil
    }
}
