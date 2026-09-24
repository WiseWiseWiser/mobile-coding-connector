import { consumeSSEStream } from './sse';

export interface ServicePortForward {
    port: number;
    label?: string;
    provider?: string;
    baseDomain?: string;
    subdomain?: string;
}

export interface ServicePortForwardStatus extends ServicePortForward {
    publicUrl?: string;
    status?: string;
    error?: string;
    active: boolean;
}

export interface ServiceStatus {
    id: string;
    name: string;
    /** 'user' for definitions you created, 'system' for server-owned subsystems. */
    kind?: 'user' | 'system' | string;
    description?: string;
    command: string;
    workingDir?: string;
    extraEnv?: Record<string, string>;
    effectivePath?: string;
    logPath: string;
    status: 'starting' | 'running' | 'stopped' | 'error' | 'unknown';
    pid: number;
    lastStartedAt?: string;
    lastExitedAt?: string;
    lastExitError?: string;
    desiredRunning: boolean;
    enabled?: boolean;
    portForward?: ServicePortForwardStatus;
    /** Remembered remote path that `service upgrade` moves an uploaded binary to. */
    upgradeTarget?: string;
    /** Steps run while the service is still running, then after it stops. */
    upgradePreStopCmds?: string[];
    upgradePostStopCmds?: string[];
    /** Per-step timeout in seconds; absent means the default, 0 disables it. */
    upgradeTimeoutSeconds?: number;
    requireAuth?: boolean;
    authUser?: string;
    authTokenMode?: 'shared' | 'custom' | string;
    authToken?: string;
    authTokens?: string[];

    /** System services only. */
    detail?: string;
    publicUrl?: string;
    port?: number;
    mocked?: boolean;
    /** System services only: the subsystem exposes an auto-start switch. */
    autoStartSwitch?: boolean;
    /** Upstream a proxying system service publishes through. */
    edge?: string;
    /** Public hostnames a system service publishes. */
    hosts?: SystemHostStatus[];
    /** Actions a system service supports ("start", "stop", "restart"). */
    actions?: string[];
}

export interface SystemHostStatus {
    host: string;
    dials: number;
    state: 'live' | 'starting' | 'dead' | 'missing' | 'unknown' | string;
}

/** A predefined template for a user service; saving it is what creates one. */
export interface ServicePreset {
    id: string;
    name: string;
    description: string;
    command: string;
    port?: number;
}

/** isSystemService reports whether the status is a server-owned subsystem. */
export function isSystemService(service: ServiceStatus): boolean {
    return service.kind === 'system';
}

export interface ServiceActionResponse {
    status: string;
    message: string;
    service: ServiceStatus;
}

export interface ServiceDefinition {
    id?: string;
    name: string;
    command: string;
    workingDir?: string;
    extraEnv?: Record<string, string>;
    portForward?: ServicePortForward;
    upgradeTarget?: string;
    upgradePreStopCmds?: string[];
    upgradePostStopCmds?: string[];
    upgradeTimeoutSeconds?: number;
    requireAuth?: boolean;
    authUser?: string;
    authTokenMode?: 'shared' | 'custom' | string;
    authToken?: string;
}

/** One upgrade step as reported by the server. */
export interface ServiceUpgradeStep {
    phase: string;
    index: number;
    total: number;
    command: string;
    exitCode: number;
    durationMs: number;
}

/** One SSE frame from the upgrade stream. */
export interface ServiceUpgradeEvent {
    type: 'section' | 'log' | 'progress' | 'done' | 'error' | string;
    message?: string;
    status?: string;
    pid?: number;
    targetPath?: string;
}

export async function fetchServices(): Promise<ServiceStatus[]> {
    const resp = await fetch('/api/services');
    if (!resp.ok) {
        throw new Error(`HTTP ${resp.status}: ${resp.statusText}`);
    }
    return await resp.json();
}

export async function fetchServicePresets(): Promise<ServicePreset[]> {
    const resp = await fetch('/api/services/presets');
    if (!resp.ok) {
        throw new Error(`HTTP ${resp.status}: ${resp.statusText}`);
    }
    return await resp.json();
}

export async function saveService(definition: ServiceDefinition): Promise<ServiceStatus> {
    const method = definition.id ? 'PUT' : 'POST';
    const resp = await fetch('/api/services', {
        method,
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify(definition),
    });
    if (!resp.ok) {
        throw new Error(await resp.text());
    }
    return await resp.json();
}

export async function deleteService(id: string): Promise<void> {
    const resp = await fetch(`/api/services?id=${encodeURIComponent(id)}`, { method: 'DELETE' });
    if (!resp.ok) {
        throw new Error(await resp.text());
    }
}

async function postServiceAction(path: string, id: string): Promise<void> {
    const resp = await fetch(`${path}?id=${encodeURIComponent(id)}`, { method: 'POST' });
    if (!resp.ok) {
        throw new Error(await resp.text());
    }
}

async function postServiceActionWithResponse(path: string, id: string): Promise<ServiceActionResponse> {
    const resp = await fetch(`${path}?id=${encodeURIComponent(id)}`, { method: 'POST' });
    if (!resp.ok) {
        throw new Error(await resp.text());
    }
    return await resp.json();
}

export async function startService(id: string): Promise<void> {
    await postServiceAction('/api/services/start', id);
}

export async function stopService(id: string): Promise<void> {
    await postServiceAction('/api/services/stop', id);
}

export async function restartService(id: string): Promise<void> {
    await postServiceAction('/api/services/restart', id);
}

export async function disableService(id: string): Promise<ServiceActionResponse> {
    return await postServiceActionWithResponse('/api/services/disable', id);
}

export async function enableService(id: string): Promise<ServiceActionResponse> {
    return await postServiceActionWithResponse('/api/services/enable', id);
}

/**
 * upgradeServiceStream runs a service upgrade and reports each phase and step
 * output line through onEvent. Resolves with the terminal `done` frame, and
 * rejects with the server's message when a step fails.
 */
export async function upgradeServiceStream(
    id: string,
    onEvent: (event: ServiceUpgradeEvent) => void,
): Promise<ServiceUpgradeEvent | undefined> {
    const resp = await fetch('/api/services/upgrade/stream', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json', Accept: 'text/event-stream' },
        body: JSON.stringify({ id }),
    });
    if (!resp.ok) {
        throw new Error(await resp.text());
    }

    let done: ServiceUpgradeEvent | undefined;
    let failure: string | undefined;

    await consumeSSEStream(resp, {
        onLog: (line) => onEvent({ type: 'log', message: line.text }),
        onError: (line) => {
            failure = line.text;
            onEvent({ type: 'error', message: line.text });
        },
        onDone: (_message, data) => {
            done = data as unknown as ServiceUpgradeEvent;
            onEvent(done);
        },
        onCustom: (data) => onEvent(data as unknown as ServiceUpgradeEvent),
    });

    if (failure) {
        throw new Error(failure);
    }
    return done;
}
