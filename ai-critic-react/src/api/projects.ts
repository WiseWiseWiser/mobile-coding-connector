// Project type from API
export interface ProjectInfo {
    id: string;
    name: string;
    repo_url: string;
    dir: string;
    ssh_key_id?: string;
    use_ssh: boolean;
    created_at: string;
}

export async function fetchProjects(): Promise<ProjectInfo[]> {
    const resp = await fetch('/api/projects');
    const data = await resp.json();
    return Array.isArray(data) ? data : [];
}

export async function deleteProject(id: string): Promise<void> {
    await fetch(`/api/projects?id=${id}`, { method: 'DELETE' });
}

export interface ProjectUpdate {
    ssh_key_id?: string | null;
    use_ssh?: boolean;
}

export async function updateProject(id: string, updates: ProjectUpdate): Promise<ProjectInfo> {
    const resp = await fetch(`/api/projects?id=${id}`, {
        method: 'PATCH',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify(updates),
    });
    if (!resp.ok) {
        const data = await resp.json();
        throw new Error(data.error || 'Failed to update project');
    }
    return resp.json();
}

// ---- Git Operations (SSE streaming) ----

const GitOps = {
    Fetch: 'fetch',
    Pull: 'pull',
    Push: 'push',
} as const;

type GitOp = typeof GitOps[keyof typeof GitOps];

export { GitOps };
export type { GitOp };

export interface GitOpRequest {
    project_id: string;
    ssh_key?: string;
}

/** Execute a git operation (fetch/pull) with SSE streaming. Returns the raw Response for SSE parsing. */
export async function runGitOp(op: GitOp, body: GitOpRequest): Promise<Response> {
    return fetch(`/api/git/${op}`, {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify(body),
    });
}

/** Execute a git operation (fetch/pull/push) with SSE streaming using project dir. */
export async function runGitOpByDir(op: GitOp, dir: string, sshKey?: string): Promise<Response> {
    const body: Record<string, string | undefined> = { dir };
    if (sshKey) {
        body.ssh_key = sshKey;
    }
    return fetch(`/api/review/${op}`, {
        method: 'POST',
        headers: {
            'Content-Type': 'application/json',
            'Accept': 'text/event-stream',
        },
        body: JSON.stringify(body),
    });
}
