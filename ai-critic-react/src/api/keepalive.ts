import { uploadFile } from './fileupload';
import type { UploadProgress } from './fileupload';

const API_BASE = '';

export interface KeepAliveStatus {
    running: boolean;
    binary_path: string;
    server_port: number;
    server_pid: number;
    keep_alive_port: number;
    keep_alive_pid: number;
    started_at?: string;
    uptime?: string;
    next_binary?: string;
    next_health_check_time?: string;
}

export interface KeepAlivePing {
    running: boolean;
    start_command?: string;
}

export interface UploadTarget {
    path: string;
    binary_name: string;
    current_version: number;
    next_version: number;
}

/** Check if the keep-alive daemon is running (local check, not proxied). */
export async function pingKeepAlive(): Promise<KeepAlivePing> {
    const res = await fetch(`${API_BASE}/api/keep-alive/ping`);
    if (!res.ok) throw new Error(`ping failed: ${res.status}`);
    return res.json();
}

/** Get the keep-alive daemon status (proxied to daemon). */
export async function getKeepAliveStatus(): Promise<KeepAliveStatus> {
    const res = await fetch(`${API_BASE}/api/keep-alive/status`);
    if (!res.ok) throw new Error(`status failed: ${res.status}`);
    return res.json();
}

/** Request the keep-alive daemon to restart the managed server. */
export async function restartServer(): Promise<{ status: string }> {
    const res = await fetch(`${API_BASE}/api/keep-alive/restart`, { method: 'POST' });
    if (!res.ok) throw new Error(`restart failed: ${res.status}`);
    return res.json();
}

/** Restart the keep-alive daemon itself with streaming logs via SSE. */
export function restartDaemonStreaming(): Promise<Response> {
    return fetch(`${API_BASE}/api/keep-alive/restart-daemon`, {
        method: 'POST',
        headers: {
            'Accept': 'text/event-stream',
        },
    });
}

/** Get the upload target path for the next binary version. */
export async function getUploadTarget(): Promise<UploadTarget> {
    const res = await fetch(`${API_BASE}/api/keep-alive/upload-target`);
    if (!res.ok) throw new Error(`upload-target failed: ${res.status}`);
    return res.json();
}

/** Notify the keep-alive daemon that a binary has been uploaded and is ready. */
export async function setBinary(path: string): Promise<{ status: string; path: string; size: number }> {
    const res = await fetch(`${API_BASE}/api/keep-alive/set-binary`, {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ path }),
    });
    if (!res.ok) throw new Error(`set-binary failed: ${res.status}`);
    return res.json();
}

/**
 * Upload a new binary using chunked upload (reuses the standard file upload).
 * 1. Gets the upload target path from the keep-alive daemon
 * 2. Uploads the file via chunked upload to the standard file upload API
 * 3. Notifies the keep-alive daemon that the binary is ready
 */
export async function uploadBinary(
    file: File,
    onProgress?: (progress: UploadProgress) => void,
): Promise<{ target: UploadTarget; path: string; size: number }> {
    // Step 1: Get the target path
    const target = await getUploadTarget();

    // Step 2: Upload via chunked upload to the standard file upload API
    const result = await uploadFile(file, target.path, onProgress);

    // Step 3: Notify keep-alive daemon
    await setBinary(target.path);

    return {
        target,
        path: result.path,
        size: result.size,
    };
}

export interface BuildableProject {
    id: string;
    name: string;
    dir: string;
    has_go_mod: boolean;
    has_build_script: boolean;
}

/** Get the list of projects that can be built from source. */
export async function getBuildableProjects(): Promise<BuildableProject[]> {
    // Use the main server's build API (not keep-alive daemon) for proper environment setup
    const res = await fetch(`${API_BASE}/api/build/buildable-projects`);
    if (!res.ok) throw new Error(`buildable-projects failed: ${res.status}`);
    return res.json();
}

export type { UploadProgress };
