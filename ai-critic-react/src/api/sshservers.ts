// SSH Servers API client

export interface SSHServer {
    id: string;
    name: string;
    host: string;
    port: number;
    username: string;
    ssh_key_id: string;
    created_at: string;
}

export async function fetchSSHServers(): Promise<SSHServer[]> {
    const resp = await fetch('/api/ssh-servers');
    if (!resp.ok) throw new Error('Failed to fetch SSH servers');
    return resp.json();
}

export async function createSSHServer(server: Omit<SSHServer, 'id' | 'created_at'>): Promise<SSHServer> {
    const resp = await fetch('/api/ssh-servers', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify(server),
    });
    if (!resp.ok) throw new Error('Failed to create SSH server');
    return resp.json();
}

export async function updateSSHServer(id: string, server: Omit<SSHServer, 'id' | 'created_at'>): Promise<SSHServer> {
    const resp = await fetch(`/api/ssh-servers/${id}`, {
        method: 'PUT',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify(server),
    });
    if (!resp.ok) throw new Error('Failed to update SSH server');
    return resp.json();
}

export async function deleteSSHServer(id: string): Promise<void> {
    const resp = await fetch(`/api/ssh-servers/${id}`, {
        method: 'DELETE',
    });
    if (!resp.ok) throw new Error('Failed to delete SSH server');
}
