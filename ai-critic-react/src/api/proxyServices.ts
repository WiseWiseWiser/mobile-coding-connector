/**
 * API client for the proxy system services' detail pages:
 * sys-http-proxy (forward proxy with Basic auth) and sys-ws-proxy (Xray
 * VMess over Cloudflare tunnel). Endpoints live in the server packages
 * httpproxy/api.go and wsproxy/api.go.
 */

export interface HttpProxyStatus {
    running: boolean;
    port?: number;
    public_url?: string;
    detail: string;
    auth_user: string;
    domain?: string;
    publish_error?: string;
    auto_start: boolean;
    uptime_secs?: number;
    configured: boolean;
}

export interface HttpProxyConfig {
    listen_port: number;
    bind_addr?: string;
    username: string;
    password: string;
    domain?: string;
    auto_start: boolean;
    allow_loopback_connect?: boolean;
}

export interface ProxyConnInfo {
    id: number;
    client: string;
    target: string;
    method: string;
    up: number;
    down: number;
    since: string;
    secs: number;
    state: string;
    err?: string;
}

export interface HttpProxyConnections {
    items: ProxyConnInfo[];
    active: number;
    connect_active: number;
    total: number;
    failures: number;
}

export interface WsProxyConnections {
    running: boolean;
    port: number;
    sessions: number;
    items: { local: string; peer: string; since?: string }[];
}

async function jsonResp<T>(resp: Response): Promise<T> {
    if (!resp.ok) {
        let msg = `${resp.status}`;
        try {
            const body = await resp.json();
            if (body && typeof body.error === 'string') msg = body.error;
            else if (body?.error?.message) msg = body.error.message;
        } catch {
            /* keep status text */
        }
        throw new Error(msg);
    }
    return resp.json() as Promise<T>;
}

export async function fetchHttpProxyStatus(): Promise<HttpProxyStatus> {
    return jsonResp(await fetch('/api/http-proxy/status'));
}

export async function startHttpProxy(): Promise<HttpProxyStatus> {
    return jsonResp(await fetch('/api/http-proxy/start', { method: 'POST' }));
}

export async function stopHttpProxy(): Promise<HttpProxyStatus> {
    return jsonResp(await fetch('/api/http-proxy/stop', { method: 'POST' }));
}

export async function fetchHttpProxyConfig(): Promise<HttpProxyConfig> {
    return jsonResp(await fetch('/api/http-proxy/config'));
}

export async function updateHttpProxyConfig(patch: Partial<HttpProxyConfig>): Promise<HttpProxyConfig> {
    return jsonResp(await fetch('/api/http-proxy/config', {
        method: 'PUT',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify(patch),
    }));
}

export async function rotateHttpProxyPassword(): Promise<{ password: string }> {
    return jsonResp(await fetch('/api/http-proxy/rotate-password', { method: 'POST' }));
}

export async function fetchHttpProxyConnections(): Promise<HttpProxyConnections> {
    return jsonResp(await fetch('/api/http-proxy/connections'));
}

export async function fetchWsProxyConnections(): Promise<WsProxyConnections> {
    return jsonResp(await fetch('/api/ws-proxy/connections'));
}

export async function fetchWsProxyVMessLink(): Promise<{ vmess_link: string }> {
    return jsonResp(await fetch('/api/ws-proxy/vmess-link'));
}

/** QR image is fetched as a blob so the auth cookie rides along. */
export async function fetchWsProxyQR(): Promise<string> {
    const resp = await fetch('/api/ws-proxy/qrcode');
    if (!resp.ok) throw new Error(`QR fetch failed: ${resp.status}`);
    const blob = await resp.blob();
    return URL.createObjectURL(blob);
}
