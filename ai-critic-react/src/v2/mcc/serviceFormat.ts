import type { ServiceStatus } from '../../api/services';

/** formatTime renders an RFC3339 timestamp for display. */
export function formatTime(value?: string): string {
    if (!value) return '';
    try {
        return new Date(value).toLocaleString();
    } catch {
        return value;
    }
}

/** appendLogLine appends to a bounded log buffer, dropping the oldest lines. */
export function appendLogLine<T>(lines: T[], next: T, max = 300): T[] {
    const merged = [...lines, next];
    if (merged.length <= max) return merged;
    return merged.slice(merged.length - max);
}

/** stringifyEnvMap renders an env map as sorted KEY=VALUE lines. */
export function stringifyEnvMap(env?: Record<string, string>): string {
    if (!env || Object.keys(env).length === 0) return '';
    return Object.entries(env)
        .sort(([a], [b]) => a.localeCompare(b))
        .map(([key, value]) => `${key}=${value}`)
        .join('\n');
}

/** parseEnvText parses KEY=VALUE lines into an env map. */
export function parseEnvText(value: string): { env?: Record<string, string>; error?: string } {
    const lines = value.split(/\r?\n/);
    const env: Record<string, string> = {};
    for (let i = 0; i < lines.length; i += 1) {
        const line = lines[i].trim();
        if (!line || line.startsWith('#')) continue;
        const idx = line.indexOf('=');
        if (idx <= 0) {
            return { error: `Invalid env on line ${i + 1}. Use KEY=VALUE.` };
        }
        const key = line.slice(0, idx).trim();
        const envValue = line.slice(idx + 1);
        if (!/^[A-Za-z_][A-Za-z0-9_]*$/.test(key)) {
            return { error: `Invalid env name on line ${i + 1}: ${key}` };
        }
        env[key] = envValue;
    }
    if (Object.keys(env).length === 0) {
        return {};
    }
    return { env };
}

/**
 * isServiceRunning reports whether a user service looks alive. System services
 * have no pid, so their status word is what counts.
 */
export function isServiceRunning(service: ServiceStatus): boolean {
    return service.pid > 0 || service.status === 'running' || service.status === 'starting';
}

export function disableMessage(service: ServiceStatus): string {
    return isServiceRunning(service)
        ? "The server won't stop immediately unless you manually stop it"
        : 'Server is already stopped';
}

export function enableMessage(service: ServiceStatus): string {
    return isServiceRunning(service)
        ? 'Server is already running'
        : "The server won't start immediately until daemon checks at next time";
}
