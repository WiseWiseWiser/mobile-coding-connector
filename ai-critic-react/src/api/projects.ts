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
