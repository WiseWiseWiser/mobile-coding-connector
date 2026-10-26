export interface TerminalSessionInfo {
    id: string;
    name: string;
    cwd: string;
    created_at: string;
    connected: boolean;
    status: 'running' | 'exited';
}

export interface TerminalSessionsResponse {
    sessions: TerminalSessionInfo[];
    page: number;
    page_size: number;
    total: number;
    total_pages: number;
}

export async function fetchTerminalSessions(page?: number, pageSize?: number): Promise<TerminalSessionInfo[]> {
    // Default restoration loads every page, not only the oldest 20 terminals.
    if (page === undefined && pageSize === undefined) {
        const sessions: TerminalSessionInfo[] = [];
        let next = 1;
        let totalPages = 1;
        do {
            const data = await fetchTerminalSessionsPaginated(next, 100);
            sessions.push(...data.sessions);
            totalPages = data.total_pages;
            next++;
        } while (next <= totalPages);
        return sessions;
    }
    const params = new URLSearchParams();
    if (page) params.set('page', page.toString());
    if (pageSize) params.set('page_size', pageSize.toString());
    
    const url = params.toString() ? `/api/terminal/sessions?${params}` : '/api/terminal/sessions';
    const resp = await fetch(url);
    if (!resp.ok) throw new Error(`Failed to load terminals (${resp.status})`);
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
    if (!resp.ok) throw new Error(`Failed to load terminals (${resp.status})`);
    const data = await resp.json();
    if (Array.isArray(data)) {
        return { sessions: data, page: 1, page_size: data.length, total: data.length, total_pages: 1 };
    }
    if (!Array.isArray(data.sessions)) throw new Error('Invalid terminal sessions response');
    return data;
}

export async function deleteTerminalSession(sessionId: string): Promise<void> {
    await fetch(`/api/terminal/sessions?id=${encodeURIComponent(sessionId)}`, { method: 'DELETE' });
}
