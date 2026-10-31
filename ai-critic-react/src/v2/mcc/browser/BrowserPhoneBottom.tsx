import type { Lens } from './browserWire';
import { nextPhoneZoom, prevPhoneZoom, zoomPercent } from './phoneZoom';

interface BrowserPhoneBottomProps {
    lens: Lens;
    zoom: number;
    keysVisible: boolean;
    onLens: (lens: Lens) => void;
    onStep: (direction: 1 | -1) => void;
    onFit: () => void;
    onKeys: () => void;
}

/**
 * The bottom island: view/ctrl, zoom detents, and the keyboard toggle.
 *
 * `⌨` only exists in CTRL because that is the only lens where typing reaches
 * the page; in VIEW the same width stays free for the zoom group.
 */
export function BrowserPhoneBottom({
    lens,
    zoom,
    keysVisible,
    onLens,
    onStep,
    onFit,
    onKeys,
}: BrowserPhoneBottomProps) {
    const canOut = prevPhoneZoom(zoom) !== null;
    const canIn = nextPhoneZoom(zoom) !== null;

    return (
        <footer className="bwp-bottom">
            <div className="bwp-seg bwp-lens" role="group" aria-label="Lens">
                <button
                    type="button"
                    data-lens="view"
                    className={lens === 'view' ? 'is-on' : ''}
                    aria-pressed={lens === 'view'}
                    onClick={() => onLens('view')}
                >
                    VIEW
                </button>
                <button
                    type="button"
                    data-lens="ctrl"
                    className={lens === 'ctrl' ? 'is-on' : ''}
                    aria-pressed={lens === 'ctrl'}
                    onClick={() => onLens('ctrl')}
                >
                    CTRL
                </button>
            </div>
            <div className="bwp-seg bwp-zoom" role="group" aria-label="Zoom">
                <button type="button" aria-label="Zoom out" disabled={!canOut} onClick={() => onStep(-1)}>
                    −
                </button>
                <button type="button" className="bwp-pct" aria-label="Fit to screen" onClick={onFit}>
                    {zoomPercent(zoom)}
                </button>
                <button type="button" aria-label="Zoom in" disabled={!canIn} onClick={() => onStep(1)}>
                    +
                </button>
            </div>
            {lens === 'ctrl' && (
                <button
                    type="button"
                    className="bwp-kb"
                    aria-label="Keyboard"
                    aria-pressed={keysVisible}
                    onClick={onKeys}
                >
                    <svg
                        viewBox="0 0 24 24"
                        width="17"
                        height="17"
                        fill="none"
                        stroke="currentColor"
                        strokeWidth="1.6"
                        strokeLinecap="round"
                        strokeLinejoin="round"
                        aria-hidden="true"
                    >
                        <rect x="2.5" y="6" width="19" height="12" rx="2.6" />
                        <path d="M6 10h.01M9.5 10h.01M13 10h.01M16.5 10h.01M6 13.5h.01M9.5 13.5h.01M13 13.5h.01M16.5 13.5h.01M8 17h8" />
                    </svg>
                </button>
            )}
        </footer>
    );
}
