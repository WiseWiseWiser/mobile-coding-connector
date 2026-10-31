import { useEffect, useRef } from 'react';
import type { KeyboardEvent } from 'react';

interface BrowserHelpProps {
    onClose: () => void;
}

/** The gestures-and-keys card behind the sheet's `?`. */
export function BrowserHelp({ onClose }: BrowserHelpProps) {
    const closeRef = useRef<HTMLButtonElement>(null);

    // Remember what opened the card so closing it returns the caret there.
    useEffect(() => {
        const opener = document.activeElement;
        closeRef.current?.focus();
        return () => {
            if (opener instanceof HTMLElement && document.contains(opener)) opener.focus();
        };
    }, []);

    // Only the close button can take focus, so Tab stays inside the card.
    const trapTab = (event: KeyboardEvent<HTMLDivElement>) => {
        if (event.key !== 'Tab') return;
        event.preventDefault();
        closeRef.current?.focus();
    };

    return (
        <div className="bw-help-layer" onClick={onClose}>
            <div
                className="bw-help-card"
                id="bw-help"
                role="dialog"
                aria-modal="true"
                aria-labelledby="bw-help-title"
                onClick={(event) => event.stopPropagation()}
                onKeyDown={trapTab}
            >
                <div className="bw-help-head">
                    <span className="bw-help-badge" aria-hidden="true">
                        ?
                    </span>
                    <div className="bw-help-titles">
                        <span className="bw-help-kicker">Browser help</span>
                        <h3 id="bw-help-title">Gestures &amp; keys</h3>
                    </div>
                    <button
                        ref={closeRef}
                        type="button"
                        className="bw-help-close"
                        onClick={onClose}
                        aria-label="Close help"
                        title="Close help"
                    >
                        ×
                    </button>
                </div>
                <dl>
                    <dt>VIEW</dt>
                    <dd>Drag to pan, double-tap to zoom to 2.5×, pinch to zoom, nothing reaches the page.</dd>
                    <dt>CTRL</dt>
                    <dd>Tap to click, drag to scroll, hold 350 ms then drag, pinch to zoom, two fingers to pan.</dd>
                    <dt>Tabs</dt>
                    <dd>Go changes this tab, + opens a new one. Pick a tab, × closes it; the last content tab cannot be closed.</dd>
                    <dt>Saved</dt>
                    <dd>
                        ☆ opens Chrome&apos;s bookmarks read-only; when Chrome is empty it links to local
                        saves. Saving the current page writes to that local list.
                    </dd>
                    <dt>Sessions</dt>
                    <dd>Every window the agent can drive; on a phone they are the chips at the top of the sheet.</dd>
                    <dt>Keys</dt>
                    <dd>Hardware keys work in CTRL; ⌨ shows the on-screen keyboard for phones.</dd>
                </dl>
                <p className="bw-help-tip">Tap outside · press Esc · or use ×</p>
            </div>
        </div>
    );
}
