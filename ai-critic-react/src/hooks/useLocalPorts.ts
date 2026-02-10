import { useState, useEffect, useRef } from 'react';
import type { LocalPortInfo } from '../api/ports';

export interface UseLocalPortsReturn {
    ports: LocalPortInfo[];
    loading: boolean;
    error: string | null;
}

export function useLocalPorts(): UseLocalPortsReturn {
    const [ports, setPorts] = useState<LocalPortInfo[]>([]);
    const [loading, setLoading] = useState(true);
    const [error, setError] = useState<string | null>(null);
    const eventSourceRef = useRef<EventSource | null>(null);

    useEffect(() => {
        const es = new EventSource('/api/ports/local/events');
        eventSourceRef.current = es;

        es.onmessage = (event) => {
            try {
                const data = JSON.parse(event.data);
                if (data.error) {
                    setError(data.error);
                    setLoading(false);
                    // Close connection for fatal errors (like missing lsof)
                    if (data.fatal) {
                        es.close();
                    }
                    return;
                }
                setPorts(data ?? []);
                setError(null);
                setLoading(false);
            } catch {
                // skip malformed data
            }
        };

        es.onerror = () => {
            // EventSource auto-reconnects; just mark error if we have no data
            // Don't show "Connection lost" if we already have a specific error (like lsof not found)
            if (loading && !error) {
                setError('Connection lost, retrying...');
            }
        };

        return () => {
            es.close();
            eventSourceRef.current = null;
        };
    }, []);

    return { ports, loading, error };
}
