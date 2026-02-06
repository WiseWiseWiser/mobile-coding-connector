export interface DiagnosticCheck {
    id: string;
    label: string;
    status: 'ok' | 'warning' | 'error';
    description: string;
}

export interface DiagnosticsData {
    overall: 'ok' | 'warning' | 'error';
    checks: DiagnosticCheck[];
}

export async function fetchDiagnostics(): Promise<DiagnosticsData> {
    const resp = await fetch('/api/ports/diagnostics');
    return resp.json();
}

export async function fetchPortLogs(port: number): Promise<string[]> {
    const resp = await fetch(`/api/ports/logs?port=${port}`);
    const data = await resp.json();
    return data ?? [];
}

export interface ProviderInfo {
    id: string;
    name: string;
    description: string;
    available: boolean;
}

export async function fetchProviders(): Promise<ProviderInfo[]> {
    const resp = await fetch('/api/ports/providers');
    const data = await resp.json();
    return data ?? [];
}

export interface PortForwardData {
    localPort: number;
    label: string;
    publicUrl: string;
    status: string;
    provider: string;
    error?: string;
}

export async function fetchPorts(): Promise<PortForwardData[]> {
    const resp = await fetch('/api/ports');
    if (!resp.ok) {
        throw new Error(`HTTP ${resp.status}: ${resp.statusText}`);
    }
    const data = await resp.json();
    return data ?? [];
}

export async function addPort(port: number, label: string, provider: string): Promise<void> {
    const resp = await fetch('/api/ports', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ port, label, provider }),
    });
    if (!resp.ok) {
        const text = await resp.text();
        throw new Error(text);
    }
}

export async function removePort(port: number): Promise<void> {
    const resp = await fetch(`/api/ports?port=${port}`, {
        method: 'DELETE',
    });
    if (!resp.ok) {
        const text = await resp.text();
        throw new Error(text);
    }
}
