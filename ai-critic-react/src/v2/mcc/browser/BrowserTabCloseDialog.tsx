import { useEffect, useRef } from 'react';
import type { PendingTabClose } from './useBrowserTabClose';

interface BrowserTabCloseDialogProps {
    pending: PendingTabClose;
    busy: boolean;
    onKeep: () => void;
    onDiscard: () => void;
    onDismiss: () => void;
}

/** A remote representation of Chrome's beforeunload prompt; never auto-accept. */
export function BrowserTabCloseDialog({ pending, busy, onKeep, onDiscard, onDismiss }: BrowserTabCloseDialogProps) {
    const layerRef = useRef<HTMLDivElement>(null);
    const safeRef = useRef<HTMLButtonElement>(null);

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
                if (!busy) { if (pending.uncertain) onDismiss(); else onKeep(); }
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
    }, [busy, pending.uncertain, onKeep, onDismiss]);

    return (
        <div className="bw-tabclose-layer" ref={layerRef}>
            <div className="bw-tabclose-card" role="dialog" aria-modal="true" aria-labelledby="bw-tabclose-title" aria-describedby="bw-tabclose-desc" aria-busy={busy}>
                <div className="bw-tabclose-kicker">Chrome tab</div>
                <h3 id="bw-tabclose-title">{pending.uncertain ? 'Check the tab in Chrome' : 'Unsaved changes?'}</h3>
                <p id="bw-tabclose-desc">
                    {pending.uncertain ? pending.uncertain : <><strong>{pending.title}</strong> is asking whether to leave. Closing it may discard unsaved changes.</>}
                </p>
                {!pending.uncertain && <p className="bw-tabclose-note">If no choice is made within a minute, we try to keep the tab open.</p>}
                <div className="bw-tabclose-actions">
                    {pending.uncertain ? (
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
