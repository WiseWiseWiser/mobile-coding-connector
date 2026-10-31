import { useLayoutEffect, useRef } from 'react';

/** Reveal the current row once per open, without taking over later scrolling. */
export function useRevealCurrent<T extends HTMLElement>(active: boolean, rowSelector: string) {
    const listRef = useRef<T>(null);
    const finished = useRef(false);
    const interacted = useRef(false);
    const timer = useRef<number | undefined>(undefined);

    useLayoutEffect(() => {
        if (!active) return;
        finished.current = false;
        interacted.current = false;
        const list = listRef.current;
        const stop = () => { interacted.current = true; };
        document.addEventListener('pointerdown', stop, true);
        document.addEventListener('keydown', stop, true);
        list?.addEventListener('touchstart', stop, { passive: true });
        list?.addEventListener('wheel', stop, { passive: true });
        return () => {
            document.removeEventListener('pointerdown', stop, true);
            document.removeEventListener('keydown', stop, true);
            list?.removeEventListener('touchstart', stop);
            list?.removeEventListener('wheel', stop);
            window.clearTimeout(timer.current);
        };
    }, [active]);

    useLayoutEffect(() => {
        const list = listRef.current;
        if (!active || finished.current || interacted.current || !list || list.clientHeight === 0) return;
        const button = list.querySelector<HTMLButtonElement>('button[aria-current="true"]');
        const row = button?.closest<HTMLElement>(rowSelector);
        if (!button || !row) return; // The current item can arrive with a later poll.
        finished.current = true;
        const view = list.getBoundingClientRect();
        const target = row.getBoundingClientRect();
        if (target.top < view.top) list.scrollTop += target.top - view.top;
        else if (target.bottom > view.bottom) list.scrollTop += target.bottom - view.bottom;
        // The sheet has a 240ms opening transform. Focusing while it is offscreen
        // can make iOS Safari pan the page despite preventScroll.
        if (!button.disabled) timer.current = window.setTimeout(() => {
            if (!interacted.current && list.contains(button) && button.getAttribute('aria-current') === 'true') {
                button.focus({ preventScroll: true });
            }
        }, 280);
    });
    return listRef;
}
