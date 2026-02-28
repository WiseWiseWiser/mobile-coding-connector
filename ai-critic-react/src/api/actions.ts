// Actions API client

export interface Action {
    id: string;
    name: string;
    icon: string;
    script: string;
}

export interface ActionRunRequest {
    project_dir: string;
    script: string;
    action_id?: string;
}

export interface LogBuffer {
    first: string[];
    last: string[];
    total: number;
}

export interface ActionStatus {
    action_id: string;
    running: boolean;
    started_at?: string;
    finished_at?: string;
    logs: LogBuffer;
    exit_code?: number;
    pid?: number;
}

export async function fetchActions(project: string): Promise<Action[]> {
    const resp = await fetch(`/api/actions?project=${encodeURIComponent(project)}`);
    if (!resp.ok) throw new Error('Failed to fetch actions');
    return resp.json();
}

export async function createAction(project: string, action: Omit<Action, 'id'>): Promise<Action> {
    const resp = await fetch(`/api/actions?project=${encodeURIComponent(project)}`, {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify(action),
    });
    if (!resp.ok) throw new Error('Failed to create action');
    return resp.json();
}

export async function updateAction(project: string, action: Action): Promise<Action> {
    const resp = await fetch(`/api/actions/${action.id}?project=${encodeURIComponent(project)}`, {
        method: 'PUT',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify(action),
    });
    if (!resp.ok) throw new Error('Failed to update action');
    return resp.json();
}

export async function deleteAction(project: string, actionId: string): Promise<void> {
    const resp = await fetch(`/api/actions/${actionId}?project=${encodeURIComponent(project)}`, {
        method: 'DELETE',
    });
    if (!resp.ok) throw new Error('Failed to delete action');
}

export async function runAction(req: ActionRunRequest): Promise<Response> {
    return fetch('/api/actions/run', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify(req),
    });
}

export async function fetchActionStatus(project: string, actionId?: string): Promise<ActionStatus | Record<string, ActionStatus>> {
    let url = `/api/actions/status?project=${encodeURIComponent(project)}`;
    if (actionId) {
        url += `&action_id=${encodeURIComponent(actionId)}`;
    }
    const resp = await fetch(url);
    if (!resp.ok) throw new Error('Failed to fetch action status');
    return resp.json();
}

export async function stopAction(project: string, actionId: string): Promise<void> {
    const resp = await fetch(`/api/actions/stop?project=${encodeURIComponent(project)}&action_id=${encodeURIComponent(actionId)}`, {
        method: 'POST',
    });
    if (!resp.ok) {
        const data = await resp.json().catch(() => ({}));
        throw new Error(data.error || 'Failed to stop action');
    }
}

export type ActionStreamHandler = {
    onLog: (message: string) => void;
    onDone?: (data: { success: string; message: string }) => void;
    onError?: (message: string) => void;
    onStatus?: (status: string) => void;
};

export function streamActionLogs(
    actionId: string,
    handlers: ActionStreamHandler
): EventSource {
    const url = `/api/actions/stream/${encodeURIComponent(actionId)}`;
    const eventSource = new EventSource(url);

    eventSource.onmessage = (event) => {
        try {
            const data = JSON.parse(event.data);
            switch (data.type) {
                case 'log':
                    handlers.onLog(data.message);
                    break;
                case 'done':
                    handlers.onDone?.(data);
                    eventSource.close();
                    break;
                case 'error':
                    handlers.onError?.(data.message);
                    eventSource.close();
                    break;
                case 'status':
                    handlers.onStatus?.(data.status);
                    break;
            }
        } catch (e) {
            console.error('Failed to parse SSE message:', e);
        }
    };

    eventSource.onerror = () => {
        handlers.onError?.('Connection error');
        eventSource.close();
    };

    return eventSource;
}
