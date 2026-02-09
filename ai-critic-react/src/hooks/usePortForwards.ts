import { useState, useEffect, useRef, useCallback } from 'react';
import { useCurrent } from './useCurrent';
import {
    fetchProviders as apiFetchProviders,
    addPort as apiAddPort,
    removePort as apiRemovePort,
} from '../api/ports';
import type { ProviderInfo as ApiProviderInfo } from '../api/ports';

// Port forward status
export const PortStatuses = {
    Active: 'active',
    Connecting: 'connecting',
    Error: 'error',
    Stopped: 'stopped',
} as const;

export type PortStatus = typeof PortStatuses[keyof typeof PortStatuses];

// Tunnel providers
export const TunnelProviders = {
    Localtunnel: 'localtunnel',
    CloudflareQuick: 'cloudflare_quick',
    CloudflareTunnel: 'cloudflare_tunnel',
    CloudflareOwned: 'cloudflare_owned',
} as const;

export type TunnelProvider = typeof TunnelProviders[keyof typeof TunnelProviders];

// Port forward type matching the backend API
export interface PortForward {
    localPort: number;
    label: string;
    publicUrl: string;
    status: PortStatus;
    provider: TunnelProvider;
    error?: string;
}

export type ProviderInfo = ApiProviderInfo;

export interface UsePortForwardsReturn {
    ports: PortForward[];
    providers: ProviderInfo[];
    loading: boolean;
    error: string | null;
    addPort: (port: number, label: string, provider?: TunnelProvider) => Promise<void>;
    removePort: (port: number) => Promise<void>;
}

export function usePortForwards(): UsePortForwardsReturn {
    const [ports, setPorts] = useState<PortForward[]>([]);
    const [providers, setProviders] = useState<ProviderInfo[]>([]);
    const [loading, setLoading] = useState(true);
    const [error, setError] = useState<string | null>(null);
    const portsRef = useCurrent(ports);
    const retryTimerRef = useRef<ReturnType<typeof setTimeout> | null>(null);

    // Fetch available providers once on mount
    useEffect(() => {
        apiFetchProviders()
            .then((data) => setProviders(data))
            .catch(() => { /* ignore provider fetch errors */ });
    }, []);

    // SSE connection for real-time port status updates
    useEffect(() => {
        let es: EventSource | null = null;
        let retryDelay = 1000;
        let cancelled = false;

        function connect() {
            if (cancelled) return;

            es = new EventSource('/api/ports/events');

            es.onmessage = (event) => {
                try {
                    const data = JSON.parse(event.data) as PortForward[];
                    setPorts(data);
                    setError(null);
                    setLoading(false);
                    retryDelay = 1000; // reset on success
                } catch {
                    // ignore parse errors
                }
            };

            es.onerror = () => {
                es?.close();
                es = null;
                if (!cancelled) {
                    // Exponential backoff reconnect, max 30s
                    retryTimerRef.current = setTimeout(connect, retryDelay);
                    retryDelay = Math.min(retryDelay * 2, 30000);
                    setError('Connection lost, reconnecting…');
                }
            };
        }

        connect();

        return () => {
            cancelled = true;
            es?.close();
            if (retryTimerRef.current) {
                clearTimeout(retryTimerRef.current);
            }
        };
    }, []);

    const addPort = useCallback(async (port: number, label: string, provider?: TunnelProvider) => {
        await apiAddPort(port, label, provider || TunnelProviders.Localtunnel);
        // SSE will push the update, no need to manually refresh
    }, []);

    const removePort = useCallback(async (port: number) => {
        await apiRemovePort(port);
        // Optimistic update - remove from local state immediately; SSE will confirm
        setPorts(portsRef.current.filter(p => p.localPort !== port));
    }, [portsRef]);

    return {
        ports,
        providers,
        loading,
        error,
        addPort,
        removePort,
    };
}
