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
