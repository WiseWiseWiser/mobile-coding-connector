import type { ServiceStatus } from '../../api/services';
import type { TunnelProvider } from '../../hooks/usePortForwards';
import { TunnelProviders } from '../../hooks/usePortForwards';
import { stringifyEnvMap } from './serviceFormat';
import type { AuthTokenMode, AuthUserMode } from './ServiceAuthFields';

export interface ServiceFormState {
    id?: string;
    name: string;
    command: string;
    workingDir: string;
    extraEnvText: string;
    /** One upgrade step per line; blank lines are ignored. */
    upgradePreStopText: string;
    upgradePostStopText: string;
    /** Per-step timeout as typed, e.g. "10m" or "0"; empty keeps the default. */
    upgradeTimeout: string;
    enablePortForward: boolean;
    port: string;
    label: string;
    provider: TunnelProvider;
    baseDomain: string;
    subdomain: string;
    requireAuth: boolean;
    authUserMode: AuthUserMode;
    authUser: string;
    authTokenMode: AuthTokenMode;
    authToken: string;
}

export function createDefaultForm(workingDir = ''): ServiceFormState {
    return {
        name: '',
        command: '',
        workingDir,
        extraEnvText: '',
        upgradePreStopText: '',
        upgradePostStopText: '',
        upgradeTimeout: '',
        enablePortForward: false,
        port: '',
        label: '',
        provider: TunnelProviders.Localtunnel,
        baseDomain: '',
        subdomain: '',
        requireAuth: false,
        authUserMode: 'any',
        authUser: '',
        authTokenMode: 'shared',
        authToken: '',
    };
}

/** toFormState seeds the edit form from an existing user service. */
export function toFormState(service: ServiceStatus): ServiceFormState {
    return {
        id: service.id,
        name: service.name,
        command: service.command,
        workingDir: service.workingDir || '',
        extraEnvText: stringifyEnvMap(service.extraEnv),
        upgradePreStopText: (service.upgradePreStopCmds || []).join('\n'),
        upgradePostStopText: (service.upgradePostStopCmds || []).join('\n'),
        upgradeTimeout: service.upgradeTimeoutSeconds === undefined ? '' : String(service.upgradeTimeoutSeconds),
        enablePortForward: !!service.portForward,
        port: service.portForward?.port ? String(service.portForward.port) : '',
        label: service.portForward?.label || '',
        provider: (service.portForward?.provider as TunnelProvider) || TunnelProviders.Localtunnel,
        baseDomain: service.portForward?.baseDomain || '',
        subdomain: service.portForward?.subdomain || '',
        requireAuth: !!service.requireAuth,
        authUserMode: service.authUser ? 'fixed' : 'any',
        authUser: service.authUser || '',
        authTokenMode: service.authTokenMode === 'custom' ? 'custom' : 'shared',
        authToken: service.authToken || '',
    };
}

/** parseUpgradeSteps turns a one-step-per-line textarea into a step list. */
export function parseUpgradeSteps(text: string): string[] {
    return text
        .split('\n')
        .map((line) => line.trim())
        .filter((line) => line.length > 0);
}

/** parseUpgradeTimeout converts a typed duration into whole seconds. */
export function parseUpgradeTimeoutInput(text: string): { seconds?: number; error?: string } {
    const trimmed = text.trim();
    if (!trimmed) {
        return {};
    }
    if (trimmed === '0') {
        return { seconds: 0 };
    }
    const match = /^(\d+(?:\.\d+)?)(s|m|h)?$/.exec(trimmed);
    if (!match) {
        return { error: `Invalid upgrade timeout ${trimmed}; use a duration like 10m, 90s, or 0 to disable.` };
    }
    const amount = Number.parseFloat(match[1]);
    const unit = match[2] || 's';
    const multiplier = unit === 'h' ? 3600 : unit === 'm' ? 60 : 1;
    const seconds = Math.ceil(amount * multiplier);
    return { seconds: seconds > 0 ? seconds : 1 };
}
