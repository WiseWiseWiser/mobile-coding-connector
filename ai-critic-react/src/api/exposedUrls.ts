// Exposed URLs API client

export interface ExposedURL {
    id: string;
    external_domain: string;
    internal_url: string;
    created_at: string;
}

export interface ExposedURLWithStatus extends ExposedURL {
    status: 'stopped' | 'connecting' | 'active' | 'error';
    tunnel_url?: string;
    error?: string;
}

export interface CloudflareStatus {
    installed: boolean;
    authenticated: boolean;
    error?: string;
}

export async function fetchExposedURLs(): Promise<ExposedURLWithStatus[]> {
    const resp = await fetch('/api/exposed-urls');
    if (!resp.ok) {
        const data = await resp.json().catch(() => ({}));
        throw new Error(data.error || 'Failed to fetch exposed URLs');
    }
    return resp.json();
}

export async function addExposedURL(externalDomain: string, internalURL: string): Promise<ExposedURLWithStatus> {
    const resp = await fetch('/api/exposed-urls/add', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ external_domain: externalDomain, internal_url: internalURL }),
    });
    if (!resp.ok) {
        const data = await resp.json().catch(() => ({}));
        throw new Error(data.error || 'Failed to add exposed URL');
    }
    return resp.json();
}

export async function updateExposedURL(id: string, externalDomain: string, internalURL: string): Promise<ExposedURLWithStatus> {
    const resp = await fetch('/api/exposed-urls/update', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ id, external_domain: externalDomain, internal_url: internalURL }),
    });
    if (!resp.ok) {
        const data = await resp.json().catch(() => ({}));
        throw new Error(data.error || 'Failed to update exposed URL');
    }
    return resp.json();
}

export async function deleteExposedURL(id: string): Promise<void> {
    const resp = await fetch('/api/exposed-urls/delete', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ id }),
    });
    if (!resp.ok) {
        const data = await resp.json().catch(() => ({}));
        throw new Error(data.error || 'Failed to delete exposed URL');
    }
}

export async function fetchExposedURLsCloudflareStatus(): Promise<CloudflareStatus> {
    const resp = await fetch('/api/exposed-urls/status');
    if (!resp.ok) {
        const data = await resp.json().catch(() => ({}));
        throw new Error(data.error || 'Failed to fetch Cloudflare status');
    }
    return resp.json();
}

export async function startExposedURLTunnel(id: string): Promise<void> {
    const resp = await fetch('/api/exposed-urls/tunnel/start', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ id }),
    });
    if (!resp.ok) {
        const data = await resp.json().catch(() => ({}));
        throw new Error(data.error || 'Failed to start tunnel');
    }
}

export async function stopExposedURLTunnel(id: string): Promise<void> {
    const resp = await fetch('/api/exposed-urls/tunnel/stop', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ id }),
    });
    if (!resp.ok) {
        const data = await resp.json().catch(() => ({}));
        throw new Error(data.error || 'Failed to stop tunnel');
    }
}
