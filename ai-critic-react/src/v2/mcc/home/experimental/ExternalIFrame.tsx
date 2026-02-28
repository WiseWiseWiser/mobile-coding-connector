import { useEffect, useRef } from 'react';

export interface ExternalIFrameProps {
    active: boolean;
    targetUrl: string;
    title: string;
    persistenceKey: string;
    className?: string;
    onLoadingChange?: (loading: boolean) => void;
    onError?: (message: string) => void;
}

type PersistentIframeEntry = {
    iframe: HTMLIFrameElement;
    host: HTMLDivElement;
};

const persistentIframes = new Map<string, PersistentIframeEntry>();

function getTargetOrigin(targetUrl: string): string {
    try {
        return new URL(targetUrl).origin;
    } catch {
        return targetUrl;
    }
}

function createPersistentIframeHost(): HTMLDivElement {
    const host = document.createElement('div');
    host.className = 'codex-web-persistent-host';
    host.style.display = 'none';
    host.style.pointerEvents = 'none';
    document.body.appendChild(host);
    return host;
}

export function ExternalIFrame({
    active,
    targetUrl,
    title,
    persistenceKey,
    className = 'codex-web-iframe-host',
    onLoadingChange,
    onError,
}: ExternalIFrameProps) {
    const containerRef = useRef<HTMLDivElement>(null);

    useEffect(() => {
        if (active) return;
        const entry = persistentIframes.get(persistenceKey);
        if (!entry) return;
        entry.host.style.display = 'none';
        entry.host.style.pointerEvents = 'none';
    }, [active, persistenceKey]);

    useEffect(() => {
        if (!active) return;
        const container = containerRef.current;
        if (!container) return;

        const targetOrigin = getTargetOrigin(targetUrl);
        let entry = persistentIframes.get(persistenceKey);
        if (!entry) {
            const host = createPersistentIframeHost();
            const iframe = document.createElement('iframe');
            iframe.className = 'codex-web-iframe';
            iframe.setAttribute('sandbox', 'allow-scripts allow-same-origin allow-forms allow-popups allow-downloads');
            iframe.src = targetUrl;
            iframe.dataset.webServiceOrigin = targetOrigin;
            host.appendChild(iframe);
            entry = { iframe, host };
            persistentIframes.set(persistenceKey, entry);
            onLoadingChange?.(true);
        } else {
            const previousOrigin = entry.iframe.dataset.webServiceOrigin;
            if (previousOrigin && previousOrigin !== targetOrigin) {
                entry.iframe.src = targetUrl;
                onLoadingChange?.(true);
            } else {
                onLoadingChange?.(false);
            }
            entry.iframe.dataset.webServiceOrigin = targetOrigin;
        }

        const iframe = entry.iframe;
        const host = entry.host;
        iframe.className = 'codex-web-iframe';
        iframe.setAttribute('title', `${title} UI`);

        const onLoad = () => onLoadingChange?.(false);
        const onFrameError = () => {
            onLoadingChange?.(false);
            onError?.(`Failed to load ${title} UI`);
        };
        iframe.addEventListener('load', onLoad);
        iframe.addEventListener('error', onFrameError);

        const syncHostRect = () => {
            const rect = container.getBoundingClientRect();
            host.style.left = `${rect.left}px`;
            host.style.top = `${rect.top}px`;
            host.style.width = `${rect.width}px`;
            host.style.height = `${rect.height}px`;
        };

        const resizeObserver = new ResizeObserver(syncHostRect);
        resizeObserver.observe(container);
        window.addEventListener('resize', syncHostRect);
        window.addEventListener('scroll', syncHostRect, true);
        syncHostRect();
        host.style.display = 'block';
        host.style.pointerEvents = 'auto';

        return () => {
            iframe.removeEventListener('load', onLoad);
            iframe.removeEventListener('error', onFrameError);
            resizeObserver.disconnect();
            window.removeEventListener('resize', syncHostRect);
            window.removeEventListener('scroll', syncHostRect, true);
            host.style.display = 'none';
            host.style.pointerEvents = 'none';
        };
    }, [active, onError, onLoadingChange, persistenceKey, targetUrl, title]);

    return <div ref={containerRef} className={className} />;
}
