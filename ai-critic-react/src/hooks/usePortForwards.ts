import { useState, useEffect, useRef } from 'react';
import { useCurrent } from './useCurrent';
import {
    fetchProviders as apiFetchProviders,
    fetchPorts as apiFetchPorts,
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
    refresh: () => void;
}

export function usePortForwards(pollIntervalMs = 3000): UsePortForwardsReturn {
    const [ports, setPorts] = useState<PortForward[]>([]);
    const [providers, setProviders] = useState<ProviderInfo[]>([]);
    const [loading, setLoading] = useState(true);
    const [error, setError] = useState<string | null>(null);
    const pollTimerRef = useRef<ReturnType<typeof setInterval> | null>(null);
    const portsRef = useCurrent(ports);

    // Fetch available providers once on mount
    useEffect(() => {
        apiFetchProviders()
            .then((data) => setProviders(data))
            .catch(() => { /* ignore provider fetch errors */ });
    }, []);

    const fetchPorts = async () => {
        try {
            const data = await apiFetchPorts();
            setPorts(data as PortForward[]);
            setError(null);
        } catch (err) {
            setError(err instanceof Error ? err.message : String(err));
        } finally {
            setLoading(false);
        }
    };

    // Initial fetch and polling
    useEffect(() => {
        fetchPorts();
        pollTimerRef.current = setInterval(fetchPorts, pollIntervalMs);
        return () => {
            if (pollTimerRef.current) {
                clearInterval(pollTimerRef.current);
            }
        };
    }, [pollIntervalMs]);

    const addPort = async (port: number, label: string, provider?: TunnelProvider) => {
        await apiAddPort(port, label, provider || TunnelProviders.Localtunnel);
        // Refresh immediately
        await fetchPorts();
    };

    const removePort = async (port: number) => {
        await apiRemovePort(port);
        // Optimistic update - remove from local state immediately
        setPorts(portsRef.current.filter(p => p.localPort !== port));
    };

    return {
        ports,
        providers,
        loading,
        error,
        addPort,
        removePort,
        refresh: fetchPorts,
    };
}
