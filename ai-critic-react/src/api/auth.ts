export async function checkAuth(): Promise<boolean> {
    const resp = await fetch('/api/auth/check');
    return resp.status !== 401;
}

export async function login(username: string, password: string): Promise<Response> {
    return fetch('/api/login', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ username, password }),
    });
}

export async function testSshKey(host: string, encryptedPrivateKey: string): Promise<{ success: boolean; output: string }> {
    const resp = await fetch('/api/ssh-keys/test', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ host, private_key: encryptedPrivateKey }),
    });
    return resp.json();
}

export interface OAuthConfigStatus {
    configured: boolean;
    client_id?: string;
}

export async function fetchOAuthConfig(): Promise<OAuthConfigStatus> {
    const resp = await fetch('/api/github/oauth-config');
    return resp.json();
}

export async function setOAuthConfig(clientId: string, clientSecret: string): Promise<Response> {
    return fetch('/api/github/oauth-config', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ client_id: clientId, client_secret: clientSecret }),
    });
}

export async function exchangeOAuthToken(code: string): Promise<{ access_token?: string; error?: string }> {
    const resp = await fetch('/api/github/oauth-token', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ code }),
    });
    return resp.json();
}

export interface GithubRepo {
    full_name: string;
    clone_url: string;
    ssh_url: string;
    html_url: string;
    description: string;
    private: boolean;
    language: string;
    updated_at: string;
}

export async function fetchGithubRepos(token: string): Promise<GithubRepo[]> {
    const resp = await fetch('/api/github/repos', {
        headers: { Authorization: `token ${token}` },
    });
    const data = await resp.json();
    return Array.isArray(data) ? data : [];
}

export async function cloneRepo(body: Record<string, unknown>): Promise<Response> {
    return fetch('/api/github/clone', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify(body),
    });
}

export async function fetchPublicKey(): Promise<string> {
    const resp = await fetch('/api/encrypt/public-key');
    if (!resp.ok) {
        throw new Error('Failed to fetch public key');
    }
    return resp.text();
}
