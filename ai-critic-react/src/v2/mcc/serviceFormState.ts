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
