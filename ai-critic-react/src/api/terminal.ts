export interface TerminalSessionInfo {
    id: string;
    name: string;
    cwd: string;
    created_at: string;
    connected: boolean;
}

export interface TerminalSessionsResponse {
    sessions: TerminalSessionInfo[];
    page: number;
    page_size: number;
    total: number;
    total_pages: number;
}

export async function fetchTerminalSessions(page?: number, pageSize?: number): Promise<TerminalSessionInfo[]> {
    const params = new URLSearchParams();
    if (page) params.set('page', page.toString());
    if (pageSize) params.set('page_size', pageSize.toString());
    
    const url = params.toString() ? `/api/terminal/sessions?${params}` : '/api/terminal/sessions';
    const resp = await fetch(url);
    const data = await resp.json();
    
    // Handle both paginated and legacy response formats
    if (data.sessions && Array.isArray(data.sessions)) {
        return data.sessions;
    }
    return Array.isArray(data) ? data : [];
}

export async function fetchTerminalSessionsPaginated(page: number = 1, pageSize: number = 20): Promise<TerminalSessionsResponse> {
    const params = new URLSearchParams({
        page: page.toString(),
        page_size: pageSize.toString(),
    });
    
    const resp = await fetch(`/api/terminal/sessions?${params}`);
    return resp.json();
}

export async function deleteTerminalSession(sessionId: string): Promise<void> {
    await fetch(`/api/terminal/sessions?id=${encodeURIComponent(sessionId)}`, { method: 'DELETE' });
}
