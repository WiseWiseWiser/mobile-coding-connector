import { useEffect, useRef } from 'react';
import type { PendingTabClose } from './useBrowserTabClose';

interface BrowserTabCloseDialogProps {
    pending: PendingTabClose;
    busy: boolean;
    onKeep: () => void;
    onDiscard: () => void;
    onDismiss: () => void;
    /** The /go warning was acknowledged; closes the page (or its window). */
    onConfirmSessionPage: () => void;
}

/**
 * Two prompts share one shell: Chrome's beforeunload question (never
 * auto-accept) and the session-page warning, which states what the close costs
 * before the tab list lets it through.
 */
export function BrowserTabCloseDialog({ pending, busy, onKeep, onDiscard, onDismiss, onConfirmSessionPage }: BrowserTabCloseDialogProps) {
    const layerRef = useRef<HTMLDivElement>(null);
    const safeRef = useRef<HTMLButtonElement>(null);
    const sessionPage = pending.sessionPage;

    useEffect(() => {
        const parent = layerRef.current?.parentElement;
        const others = Array.from(parent?.children ?? []).filter((node): node is HTMLElement => node instanceof HTMLElement && node !== layerRef.current);
        const prior = others.map((node) => node.inert);
        others.forEach((node) => { node.inert = true; });
        safeRef.current?.focus();
        return () => {
            others.forEach((node, index) => { node.inert = prior[index]; });
            if (pending.opener?.isConnected) pending.opener.focus({ preventScroll: true });
        };
    }, [pending.opener]);

    useEffect(() => { if (pending.uncertain) safeRef.current?.focus(); }, [pending.uncertain]);

    useEffect(() => {
        const onKey = (event: KeyboardEvent) => {
            event.stopPropagation();
            if (event.key === 'Escape') {
                event.preventDefault(); event.stopPropagation();
                if (busy) return;
                if (pending.uncertain || pending.sessionPage) onDismiss(); else onKeep();
            } else if (event.key === 'Tab') {
                const buttons = layerRef.current?.querySelectorAll<HTMLButtonElement>('button:not(:disabled)');
                if (!buttons?.length) return;
                const first = buttons[0], last = buttons[buttons.length - 1];
                if (event.shiftKey && document.activeElement === first) { event.preventDefault(); last.focus(); }
                else if (!event.shiftKey && document.activeElement === last) { event.preventDefault(); first.focus(); }
            }
        };
        document.addEventListener('keydown', onKey);
        return () => document.removeEventListener('keydown', onKey);
    }, [busy, pending.uncertain, pending.sessionPage, onKeep, onDismiss]);

    return (
        <div className="bw-tabclose-layer" ref={layerRef}>
            <div className="bw-tabclose-card" role="dialog" aria-modal="true" aria-labelledby="bw-tabclose-title" aria-describedby="bw-tabclose-desc" aria-busy={busy}>
                <div className={sessionPage ? 'bw-tabclose-kicker is-warn' : 'bw-tabclose-kicker'}>
                    {sessionPage ? 'Session control page' : 'Chrome tab'}
                </div>
                <h3 id="bw-tabclose-title">
                    {sessionPage ? sessionPage.title : pending.uncertain ? 'Check the tab in Chrome' : 'Unsaved changes?'}
                </h3>
                <p id="bw-tabclose-desc">
                    {sessionPage ? sessionPage.body : pending.uncertain ? pending.uncertain : <><strong>{pending.title}</strong> is asking whether to leave. Closing it may discard unsaved changes.</>}
                </p>
                {sessionPage?.anchor && <p className="bw-tabclose-anchor"><code>{sessionPage.anchor}</code></p>}
                {sessionPage && <p className="bw-tabclose-note">{sessionPage.note}</p>}
                {!sessionPage && !pending.uncertain && <p className="bw-tabclose-note">If no choice is made within a minute, we try to keep the tab open.</p>}
                <div className="bw-tabclose-actions">
                    {sessionPage ? (
                        <>
                            <button ref={safeRef} type="button" onClick={onDismiss} disabled={busy}>Cancel</button>
                            <button className="is-danger" type="button" onClick={onConfirmSessionPage} disabled={busy}>
                                {busy ? 'Working…' : sessionPage.confirm}
                            </button>
                        </>
                    ) : pending.uncertain ? (
                        <button ref={safeRef} type="button" onClick={onDismiss}>Dismiss</button>
                    ) : (
                        <>
                            <button ref={safeRef} type="button" onClick={onKeep} disabled={busy}>Keep tab</button>
                            <button className="is-danger" type="button" onClick={onDiscard} disabled={busy}>{busy ? 'Working…' : 'Discard and close'}</button>
                        </>
                    )}
                </div>
            </div>
        </div>
    );
}
