// Checkpoint API client

export interface CheckpointSummary {
    id: number;
    name: string;
    timestamp: string;
    file_count: number;
}

export interface ChangedFile {
    path: string;
    status: string; // "added" | "modified" | "deleted"
}

export interface CheckpointDetail {
    id: number;
    name: string;
    timestamp: string;
    files: ChangedFile[];
}

export interface CreateCheckpointRequest {
    project_dir: string;
    name?: string;
    message?: string;
    file_paths: string[];
}

export async function fetchCheckpoints(project: string): Promise<CheckpointSummary[]> {
    const resp = await fetch(`/api/checkpoints?project=${encodeURIComponent(project)}`);
    if (!resp.ok) throw new Error('Failed to fetch checkpoints');
    return resp.json();
}

export async function createCheckpoint(project: string, req: CreateCheckpointRequest): Promise<CheckpointSummary> {
    const resp = await fetch(`/api/checkpoints?project=${encodeURIComponent(project)}`, {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify(req),
    });
    if (!resp.ok) throw new Error('Failed to create checkpoint');
    return resp.json();
}

export async function fetchCheckpointDetail(project: string, id: number): Promise<CheckpointDetail> {
    const resp = await fetch(`/api/checkpoints/${id}?project=${encodeURIComponent(project)}`);
    if (!resp.ok) throw new Error('Failed to fetch checkpoint detail');
    return resp.json();
}

export async function deleteCheckpoint(project: string, id: number): Promise<void> {
    const resp = await fetch(`/api/checkpoints/${id}?project=${encodeURIComponent(project)}`, {
        method: 'DELETE',
    });
    if (!resp.ok) throw new Error('Failed to delete checkpoint');
}

export async function fetchCurrentChanges(project: string, projectDir: string): Promise<ChangedFile[]> {
    const resp = await fetch(`/api/checkpoints/current?project=${encodeURIComponent(project)}&project_dir=${encodeURIComponent(projectDir)}`);
    if (!resp.ok) throw new Error('Failed to fetch current changes');
    return resp.json();
}

// --- Diff API ---

export interface DiffLine {
    type: string; // "context" | "add" | "delete"
    content: string;
    old_num?: number;
    new_num?: number;
}

export interface DiffHunk {
    old_start: number;
    old_lines: number;
    new_start: number;
    new_lines: number;
    lines: DiffLine[];
}

export interface FileDiff {
    path: string;
    status: string;
    hunks: DiffHunk[];
}

export async function fetchCheckpointDiff(project: string, id: number): Promise<FileDiff[]> {
    const resp = await fetch(`/api/checkpoints/${id}/diff?project=${encodeURIComponent(project)}`);
    if (!resp.ok) throw new Error('Failed to fetch diff');
    return resp.json();
}

export async function fetchCurrentDiff(project: string, projectDir: string): Promise<FileDiff[]> {
    const resp = await fetch(`/api/checkpoints/current/diff?project=${encodeURIComponent(project)}&project_dir=${encodeURIComponent(projectDir)}`);
    if (!resp.ok) throw new Error('Failed to fetch current diff');
    return resp.json();
}

// --- File Browser API ---

export interface FileEntry {
    name: string;
    path: string;
    is_dir: boolean;
    size?: number;
}

export async function fetchFiles(projectDir: string, path?: string): Promise<FileEntry[]> {
    let url = `/api/files?project_dir=${encodeURIComponent(projectDir)}&hidden=true`;
    if (path) url += `&path=${encodeURIComponent(path)}`;
    const resp = await fetch(url);
    if (!resp.ok) throw new Error('Failed to fetch files');
    return resp.json();
}

export async function fetchFileContent(projectDir: string, path: string): Promise<string> {
    const resp = await fetch(`/api/files/content?project_dir=${encodeURIComponent(projectDir)}&path=${encodeURIComponent(path)}`);
    if (!resp.ok) throw new Error('Failed to fetch file content');
    const data = await resp.json();
    return data.content;
}

export async function fetchHomeDir(): Promise<string> {
    const resp = await fetch('/api/files/home');
    if (!resp.ok) throw new Error('Failed to fetch home directory');
    const data = await resp.json();
    return data.home_dir;
}

export async function fetchServerFiles(basePath: string, path?: string): Promise<FileEntry[]> {
    let url = `/api/server/files?base_path=${encodeURIComponent(basePath)}`;
    if (path) url += `&path=${encodeURIComponent(path)}`;
    const resp = await fetch(url);
    if (!resp.ok) throw new Error('Failed to fetch server files');
    return resp.json();
}
