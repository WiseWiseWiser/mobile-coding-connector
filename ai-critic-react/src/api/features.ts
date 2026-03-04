export interface Feature {
    id: string;
    title: string;
    description?: string;
    status: string;
    created_at: number;
}

export async function fetchFeatures(projectName: string): Promise<Feature[]> {
    const resp = await fetch(`/api/features?project=${encodeURIComponent(projectName)}`);
    if (!resp.ok) throw new Error(await resp.text());
    return resp.json();
}

export async function createFeature(projectName: string, title: string, description: string): Promise<Feature> {
    const resp = await fetch(`/api/features?project=${encodeURIComponent(projectName)}`, {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ title, description }),
    });
    if (!resp.ok) throw new Error(await resp.text());
    return resp.json();
}

export async function updateFeature(projectName: string, featureId: string, updates: { title?: string; description?: string }): Promise<Feature> {
    const resp = await fetch(`/api/features?project=${encodeURIComponent(projectName)}&id=${encodeURIComponent(featureId)}`, {
        method: 'PATCH',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify(updates),
    });
    if (!resp.ok) throw new Error(await resp.text());
    return resp.json();
}

export async function deleteFeature(projectName: string, featureId: string): Promise<void> {
    const resp = await fetch(`/api/features?project=${encodeURIComponent(projectName)}&id=${encodeURIComponent(featureId)}`, {
        method: 'DELETE',
    });
    if (!resp.ok) throw new Error(await resp.text());
}
