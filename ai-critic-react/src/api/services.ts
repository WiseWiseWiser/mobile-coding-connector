export interface ServicePortForward {
    port: number;
    label?: string;
    provider?: string;
    baseDomain?: string;
    subdomain?: string;
}

export interface ServicePortForwardStatus extends ServicePortForward {
    publicUrl?: string;
    status?: string;
    error?: string;
    active: boolean;
}

export interface ServiceStatus {
    id: string;
    name: string;
    command: string;
    projectDir?: string;
    logPath: string;
    status: 'starting' | 'running' | 'stopped' | 'error';
    pid: number;
    lastStartedAt?: string;
    lastExitedAt?: string;
    lastExitError?: string;
    desiredRunning: boolean;
    portForward?: ServicePortForwardStatus;
}

export interface ServiceDefinition {
    id?: string;
    name: string;
    command: string;
    projectDir?: string;
    portForward?: ServicePortForward;
}

export async function fetchServices(projectDir?: string): Promise<ServiceStatus[]> {
    const url = new URL('/api/services', window.location.origin);
    if (projectDir) {
        url.searchParams.set('project_dir', projectDir);
    }
    const resp = await fetch(url.pathname + url.search);
    if (!resp.ok) {
        throw new Error(`HTTP ${resp.status}: ${resp.statusText}`);
    }
    return await resp.json();
}

export async function saveService(definition: ServiceDefinition): Promise<ServiceStatus> {
    const method = definition.id ? 'PUT' : 'POST';
    const resp = await fetch('/api/services', {
        method,
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify(definition),
    });
    if (!resp.ok) {
        throw new Error(await resp.text());
    }
    return await resp.json();
}

export async function deleteService(id: string): Promise<void> {
    const resp = await fetch(`/api/services?id=${encodeURIComponent(id)}`, { method: 'DELETE' });
    if (!resp.ok) {
        throw new Error(await resp.text());
    }
}

async function postServiceAction(path: string, id: string): Promise<void> {
    const resp = await fetch(`${path}?id=${encodeURIComponent(id)}`, { method: 'POST' });
    if (!resp.ok) {
        throw new Error(await resp.text());
    }
}

export async function startService(id: string): Promise<void> {
    await postServiceAction('/api/services/start', id);
}

export async function stopService(id: string): Promise<void> {
    await postServiceAction('/api/services/stop', id);
}

export async function restartService(id: string): Promise<void> {
    await postServiceAction('/api/services/restart', id);
}
