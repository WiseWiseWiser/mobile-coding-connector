// Cloudflare settings API client

export interface CertFileInfo {
    name: string;
    path: string;
    size: number;
}

export interface CloudflareStatus {
    installed: boolean;
    authenticated: boolean;
    error?: string;
    cert_files?: CertFileInfo[];
}

export interface TunnelInfo {
    id: string;
    name: string;
    created_at?: string;
    connections?: unknown[];
}

export async function fetchCloudflareStatus(): Promise<CloudflareStatus> {
    const resp = await fetch('/api/cloudflare/status');
    if (!resp.ok) throw new Error('Failed to fetch cloudflare status');
    return resp.json();
}

/** Start cloudflared login, returns raw Response for SSE streaming. */
export async function cloudflareLogin(): Promise<Response> {
    const resp = await fetch('/api/cloudflare/login', { method: 'POST' });
    if (!resp.ok) {
        const text = await resp.text();
        throw new Error(text || 'Login failed');
    }
    return resp;
}

export async function fetchTunnels(): Promise<TunnelInfo[]> {
    const resp = await fetch('/api/cloudflare/tunnels');
    if (!resp.ok) {
        const text = await resp.text();
        throw new Error(text || 'Failed to fetch tunnels');
    }
    return resp.json();
}

export async function createTunnel(name: string): Promise<{ message: string }> {
    const resp = await fetch(`/api/cloudflare/tunnels?name=${encodeURIComponent(name)}`, {
        method: 'POST',
    });
    if (!resp.ok) {
        const text = await resp.text();
        throw new Error(text || 'Failed to create tunnel');
    }
    return resp.json();
}

export async function deleteTunnel(name: string): Promise<{ message: string }> {
    const resp = await fetch(`/api/cloudflare/tunnels?name=${encodeURIComponent(name)}`, {
        method: 'DELETE',
    });
    if (!resp.ok) {
        const text = await resp.text();
        throw new Error(text || 'Failed to delete tunnel');
    }
    return resp.json();
}

// ---- Owned Domains ----

export async function fetchOwnedDomains(): Promise<string[]> {
    const resp = await fetch('/api/cloudflare/owned-domains');
    if (!resp.ok) throw new Error('Failed to fetch owned domains');
    const data = await resp.json();
    return data.owned_domains || [];
}

export async function saveOwnedDomains(domains: string[]): Promise<void> {
    const resp = await fetch('/api/cloudflare/owned-domains', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ owned_domains: domains }),
    });
    if (!resp.ok) {
        const text = await resp.text();
        throw new Error(text || 'Failed to save owned domains');
    }
}
