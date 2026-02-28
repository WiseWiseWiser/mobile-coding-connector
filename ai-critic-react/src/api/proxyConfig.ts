// Proxy server configuration API client

export interface ProxyServer {
    id: string;
    name: string;
    host: string;
    port: number;
    protocol?: string; // http, https, socks5 (default: http)
    username?: string;
    password?: string;
    domains: string[];
}

export interface ProxyConfig {
    enabled: boolean;
    servers: ProxyServer[];
}

export async function fetchProxyConfig(): Promise<ProxyConfig> {
    const resp = await fetch('/api/proxy/config');
    if (!resp.ok) {
        const data = await resp.json().catch(() => ({}));
        throw new Error(data.error || 'Failed to load proxy config');
    }
    const config = await resp.json();
    
    // Fetch servers separately
    const serversResp = await fetch('/api/proxy/servers');
    if (!serversResp.ok) {
        const data = await serversResp.json().catch(() => ({}));
        throw new Error(data.error || 'Failed to load proxy servers');
    }
    
    config.servers = await serversResp.json();
    return config;
}

export async function saveProxyConfig(config: ProxyConfig): Promise<void> {
    // Save global enabled state
    const configResp = await fetch('/api/proxy/config', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ enabled: config.enabled }),
    });
    if (!configResp.ok) {
        const data = await configResp.json().catch(() => ({}));
        throw new Error(data.error || 'Failed to save proxy config');
    }
    
    // Save servers
    const serversResp = await fetch('/api/proxy/servers', {
        method: 'PUT',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify(config.servers),
    });
    if (!serversResp.ok) {
        const data = await serversResp.json().catch(() => ({}));
        throw new Error(data.error || 'Failed to save proxy servers');
    }
}

export function generateProxyServerId(): string {
    return 'proxy_' + Date.now() + '_' + Math.random().toString(36).substr(2, 9);
}
