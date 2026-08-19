import Foundation

/// One linked worktree from GET /api/wrk/projects.
public struct WrkWorktreeStatus: Decodable, Identifiable, Equatable {
    public var id: String { path }
    public let path: String
    public let name: String
    public let branch: String?
    public let clean: Bool
    public let isMain: Bool
    public let error: String?

    enum CodingKeys: String, CodingKey {
        case path, name, branch, clean, error
        case isMain = "is_main"
    }

    public init(
        path: String,
        name: String,
        branch: String? = nil,
        clean: Bool = true,
        isMain: Bool = false,
        error: String? = nil
    ) {
        self.path = path
        self.name = name
        self.branch = branch
        self.clean = clean
        self.isMain = isMain
        self.error = error
    }
}

/// One wrk-registered project.
public struct WrkProjectStatus: Decodable, Identifiable, Equatable {
    public var id: String { path }
    public let path: String
    public let name: String
    public let branch: String?
    public let commit: String?
    public let subject: String?
    public let clean: Bool
    public let error: String?
    public let worktrees: [WrkWorktreeStatus]?

    public init(
        path: String,
        name: String,
        branch: String? = nil,
        commit: String? = nil,
        subject: String? = nil,
        clean: Bool = true,
        error: String? = nil,
        worktrees: [WrkWorktreeStatus]? = nil
    ) {
        self.path = path
        self.name = name
        self.branch = branch
        self.commit = commit
        self.subject = subject
        self.clean = clean
        self.error = error
        self.worktrees = worktrees
    }
}

public struct WrkListProjectsResponse: Decodable {
    public let projects: [WrkProjectStatus]

    public init(projects: [WrkProjectStatus] = []) {
        self.projects = projects
    }
}

public struct WrkCreateWorktreeRequest: Encodable {
    public let projectPath: String
    public let task: String?

    enum CodingKeys: String, CodingKey {
        case projectPath = "project_path"
        case task
    }

    public init(projectPath: String, task: String? = nil) {
        self.projectPath = projectPath
        self.task = task
    }
}

public struct WrkCreateWorktreeResponse: Decodable {
    public let path: String
    public let branch: String

    public init(path: String, branch: String) {
        self.path = path
        self.branch = branch
    }
}
