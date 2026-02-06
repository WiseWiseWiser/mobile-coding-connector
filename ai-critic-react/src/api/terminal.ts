export interface TerminalSessionInfo {
    id: string;
    name: string;
    cwd: string;
    created_at: string;
    connected: boolean;
}

export async function fetchTerminalSessions(): Promise<TerminalSessionInfo[]> {
    const resp = await fetch('/api/terminal/sessions');
    const data = await resp.json();
    return Array.isArray(data) ? data : [];
}

export async function deleteTerminalSession(sessionId: string): Promise<void> {
    await fetch(`/api/terminal/sessions?id=${encodeURIComponent(sessionId)}`, { method: 'DELETE' });
}
