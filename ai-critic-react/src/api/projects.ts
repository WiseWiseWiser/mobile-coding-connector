export interface Todo {
    id: string;
    text: string;
    done: boolean;
    created_at: string;
    updated_at?: string;
}

// Worktree configuration stored in project
export interface WorktreeConfig {
    [id: string]: {
        path: string;
        branch: string;
    };
}

// Project type from API
export interface ProjectInfo {
    id: string;
    name: string;
    repo_url: string;
    dir: string;
    ssh_key_id?: string;
    use_ssh: boolean;
    created_at: string;
    dir_exists: boolean;
    git_status?: {
        is_clean: boolean;
        uncommitted: number;
    };
    parent_id?: string;
    todos?: Todo[];
    worktrees?: WorktreeConfig;
    readme?: string;
}

export async function fetchProjects(options?: { all?: boolean; parentId?: string }): Promise<ProjectInfo[]> {
    const params = new URLSearchParams();
    if (options?.all) {
        params.set('all', 'true');
    }
    if (options?.parentId !== undefined) {
        params.set('parent_id', options.parentId);
    }
    const query = params.toString();
    const resp = await fetch(`/api/projects${query ? `?${query}` : ''}`);
    const data = await resp.json();
    return Array.isArray(data) ? data : [];
}

export async function fetchSubProjects(parentId: string): Promise<ProjectInfo[]> {
    const resp = await fetch(`/api/projects?parent_id=${encodeURIComponent(parentId)}`);
    const data = await resp.json();
    return Array.isArray(data) ? data : [];
}

export async function deleteProject(id: string): Promise<void> {
    await fetch(`/api/projects?id=${id}`, { method: 'DELETE' });
}

export interface AddProjectRequest {
    name?: string;
    dir: string;
    parent_id?: string;
}

export interface AddProjectResponse {
    status: string;
    id: string;
    dir: string;
    name: string;
    error?: string;
}

export async function addProject(req: AddProjectRequest): Promise<AddProjectResponse> {
    const resp = await fetch('/api/projects', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify(req),
    });
    const data = await resp.json();
    if (!resp.ok) {
        throw new Error(data.error || 'Failed to add project');
    }
    return data;
}

export interface ProjectUpdate {
    ssh_key_id?: string | null;
    use_ssh?: boolean;
    parent_id?: string | null;
    worktrees?: WorktreeConfig;
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

// ---- Todo Operations ----

export async function fetchTodos(projectId: string): Promise<Todo[]> {
    console.log('[fetchTodos] Loading todos for project:', projectId);
    try {
        const resp = await fetch(`/api/projects/todos?project_id=${projectId}`);
        if (!resp.ok) {
            console.warn('[fetchTodos] Failed to fetch todos, status:', resp.status);
            return [];
        }
        const data = await resp.json();
        console.log('[fetchTodos] Received data:', data);
        return Array.isArray(data) ? data : [];
    } catch (err) {
        console.error('[fetchTodos] Error:', err);
        return [];
    }
}

export async function addTodo(projectId: string, text: string): Promise<Todo> {
    console.log('[addTodo] Adding todo for project:', projectId, 'text:', text);
    try {
        const resp = await fetch(`/api/projects/todos?project_id=${projectId}`, {
            method: 'POST',
            headers: { 'Content-Type': 'application/json' },
            body: JSON.stringify({ text }),
        });
        if (!resp.ok) {
            const data = await resp.json();
            throw new Error(data.error || 'Failed to add todo');
        }
        const todo = await resp.json();
        console.log('[addTodo] Added todo successfully:', todo);
        return todo;
    } catch (err) {
        console.error('[addTodo] Error:', err);
        throw err;
    }
}

export async function updateTodo(projectId: string, todoId: string, updates: { text?: string; done?: boolean }): Promise<Todo> {
    console.log('[updateTodo] Updating todo:', todoId, 'for project:', projectId, 'updates:', updates);
    try {
        const resp = await fetch(`/api/projects/todos?project_id=${projectId}`, {
            method: 'PUT',
            headers: { 'Content-Type': 'application/json' },
            body: JSON.stringify({ id: todoId, ...updates }),
        });
        if (!resp.ok) {
            const data = await resp.json();
            throw new Error(data.error || 'Failed to update todo');
        }
        const todo = await resp.json();
        console.log('[updateTodo] Updated todo successfully:', todo);
        return todo;
    } catch (err) {
        console.error('[updateTodo] Error:', err);
        throw err;
    }
}

export async function deleteTodo(projectId: string, todoId: string): Promise<void> {
    console.log('[deleteTodo] Deleting todo:', todoId, 'for project:', projectId);
    try {
        const resp = await fetch(`/api/projects/todos?project_id=${projectId}&todo_id=${todoId}`, {
            method: 'DELETE',
        });
        if (!resp.ok) {
            const data = await resp.json();
            throw new Error(data.error || 'Failed to delete todo');
        }
        console.log('[deleteTodo] Deleted todo successfully');
    } catch (err) {
        console.error('[deleteTodo] Error:', err);
        throw err;
    }
}

// ---- README Operations ----

export async function fetchReadme(projectId: string): Promise<string> {
    console.log('[fetchReadme] Loading readme for project:', projectId);
    try {
        const resp = await fetch(`/api/projects/readme?project_id=${projectId}`);
        if (!resp.ok) {
            console.warn('[fetchReadme] Failed to fetch readme, status:', resp.status);
            return '';
        }
        const data = await resp.json();
        return data.readme || '';
    } catch (err) {
        console.error('[fetchReadme] Error:', err);
        return '';
    }
}

export async function updateReadme(projectId: string, readme: string): Promise<void> {
    console.log('[updateReadme] Updating readme for project:', projectId);
    try {
        const resp = await fetch(`/api/projects/readme?project_id=${projectId}`, {
            method: 'PUT',
            headers: { 'Content-Type': 'application/json' },
            body: JSON.stringify({ readme }),
        });
        if (!resp.ok) {
            const data = await resp.json();
            throw new Error(data.error || 'Failed to update readme');
        }
        console.log('[updateReadme] Updated readme successfully');
    } catch (err) {
        console.error('[updateReadme] Error:', err);
        throw err;
    }
}
