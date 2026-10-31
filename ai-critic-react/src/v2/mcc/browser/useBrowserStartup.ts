import { useRef, useState } from 'react';
import { startBrowser } from './browserApi';

/** A local guard complements the server's cross-viewer startup lock. */
export function useBrowserStartup(onStarted: () => Promise<void>) {
    const inFlight = useRef(false);
    const [busy, setBusy] = useState(false);
    const [error, setError] = useState('');
    const start = async () => {
        if (inFlight.current) return;
        inFlight.current = true;
        setBusy(true);
        setError('');
        try {
            await startBrowser();
            await onStarted();
        } catch (err) {
            const detail = err instanceof Error ? err.message : String(err);
            setError(/failed to fetch|networkerror|load failed/i.test(detail)
                ? `${detail}. Chrome may still be starting; check it before retrying.`
                : detail);
            void onStarted();
        } finally {
            inFlight.current = false;
            setBusy(false);
        }
    };
    return { start, busy, error };
}
