export interface ParsedProxyUrl {
    protocol: string;
    host: string;
    port: string;
    username: string;
    password: string;
}

export function parseProxyUrl(raw: string): ParsedProxyUrl {
    const normalized = /^[a-z]+:\/\//i.test(raw) ? raw : `http://${raw}`;
    const parsed = new URL(normalized);
    const protocol = parsed.protocol.replace(':', '');

    return {
        protocol: protocol === 'socks5' ? 'socks5' : protocol,
        host: parsed.hostname,
        port: parsed.port,
        username: parsed.username ? decodeURIComponent(parsed.username) : '',
        password: parsed.password ? decodeURIComponent(parsed.password) : '',
    };
}
