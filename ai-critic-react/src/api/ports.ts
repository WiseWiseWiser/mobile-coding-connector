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

export async function fetchDomainHealthLogs(domain: string): Promise<string[]> {
    const resp = await fetch(`/api/domains/health-logs?domain=${encodeURIComponent(domain)}`);
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

export interface AddPortRequest {
    port: number;
    label: string;
    provider: string;
    baseDomain?: string;
    subdomain?: string;
}

export async function addPort(req: AddPortRequest): Promise<void> {
    const resp = await fetch('/api/ports', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify(req),
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

export interface LocalPortInfo {
    port: number;
    pid: number;
    ppid: number;
    command: string;
    cmdline: string;
}

// Port mapping names API

export interface PortMappingName {
    port: string;
    domain: string;
}

export type PortMappingNames = Record<string, string>;

export async function fetchPortMappingName(port: number): Promise<string | null> {
    const resp = await fetch(`/api/ports/mapping-names?port=${port}`);
    if (!resp.ok) {
        if (resp.status === 404) return null;
        throw new Error(`HTTP ${resp.status}: ${resp.statusText}`);
    }
    const data = await resp.json() as PortMappingName;
    return data.domain || null;
}

export async function fetchAllPortMappingNames(): Promise<PortMappingNames> {
    const resp = await fetch('/api/ports/mapping-names');
    if (!resp.ok) {
        throw new Error(`HTTP ${resp.status}: ${resp.statusText}`);
    }
    return await resp.json() as PortMappingNames;
}

export interface SavePortMappingNameRequest {
    port: number;
    domain: string;
}

export async function savePortMappingName(req: SavePortMappingNameRequest): Promise<void> {
    const resp = await fetch('/api/ports/mapping-names', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify(req),
    });
    if (!resp.ok) {
        const text = await resp.text();
        throw new Error(text);
    }
}

export async function deletePortMappingName(port: number): Promise<void> {
    const resp = await fetch(`/api/ports/mapping-names?port=${port}`, {
        method: 'DELETE',
    });
    if (!resp.ok) {
        const text = await resp.text();
        throw new Error(text);
    }
}

export async function killProcess(pid: number, port?: number): Promise<void> {
    let url = `/api/ports/local/kill?pid=${pid}`;
    if (port !== undefined) {
        url += `&port=${port}`;
    }
    const resp = await fetch(url, {
        method: 'POST',
    });
    if (!resp.ok) {
        const text = await resp.text();
        throw new Error(text);
    }
}

export async function fetchProtectedPorts(): Promise<number[]> {
    const resp = await fetch('/api/ports/protected');
    if (!resp.ok) {
        const text = await resp.text();
        throw new Error(text);
    }
    const data = await resp.json();
    return data.protected_ports || [];
}

export async function addProtectedPort(port: number): Promise<void> {
    const resp = await fetch(`/api/ports/protected?port=${port}`, {
        method: 'POST',
    });
    if (!resp.ok) {
        const text = await resp.text();
        throw new Error(text);
    }
}

export async function removeProtectedPort(port: number): Promise<void> {
    const resp = await fetch(`/api/ports/protected?port=${port}`, {
        method: 'DELETE',
    });
    if (!resp.ok) {
        const text = await resp.text();
        throw new Error(text);
    }
}
