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
