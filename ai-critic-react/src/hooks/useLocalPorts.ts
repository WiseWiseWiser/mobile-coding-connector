import { useState, useEffect, useCallback } from 'react';
import { fetchLocalPorts, type LocalPortInfo } from '../api/ports';

export interface UseLocalPortsReturn {
    ports: LocalPortInfo[];
    loading: boolean;
    error: string | null;
    refresh: () => Promise<void>;
}

export function useLocalPorts(): UseLocalPortsReturn {
    const [ports, setPorts] = useState<LocalPortInfo[]>([]);
    const [loading, setLoading] = useState(true);
    const [error, setError] = useState<string | null>(null);

    const refresh = useCallback(async () => {
        try {
            setLoading(true);
            setError(null);
            const data = await fetchLocalPorts();
            setPorts(data);
        } catch (err) {
            setError(err instanceof Error ? err.message : 'Failed to fetch local ports');
        } finally {
            setLoading(false);
        }
    }, []);

    useEffect(() => {
        refresh();
        // Refresh every 5 seconds
        const timer = setInterval(refresh, 5000);
        return () => clearInterval(timer);
    }, [refresh]);

    return {
        ports,
        loading,
        error,
        refresh,
    };
}
