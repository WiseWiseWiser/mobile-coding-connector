import type { Lens } from './browserWire';

export interface BrowserDockProps {
    lens: Lens;
    zoom: number;
    readOnly: boolean;
    keysVisible: boolean;
    onLens: (lens: Lens) => void;
    onKeys: () => void;
    onHelp: () => void;
}

const LENSES: { id: Lens; label: string; title: string }[] = [
    { id: 'view', label: 'VIEW', title: 'Watch and zoom; nothing reaches the page' },
    { id: 'ctrl', label: 'CTRL', title: 'Tap, scroll and type into the page' },
];

/**
 * Bottom bar: which lens the pointer is in, the current zoom, and the two
 * overlays (keyboard, help).
 *
 * CTRL is the only lens that sends input, so a stray thumb cannot click a button
 * on the remote page while somebody is just looking.
 */
export function BrowserDock(props: BrowserDockProps) {
    return (
        <div className="bw-dock">
            <div className="bw-lens" role="group" aria-label="Input mode">
                {LENSES.map((lens) => (
                    <button
                        key={lens.id}
                        type="button"
                        className={`bw-lensbtn${props.lens === lens.id ? ' is-on' : ''}`}
                        data-lens={lens.id}
                        aria-pressed={props.lens === lens.id}
                        onClick={() => props.onLens(lens.id)}
                        title={lens.title}
                    >
                        {lens.label}
                    </button>
                ))}
            </div>
            <span className="bw-zoomread">{Math.round(props.zoom * 100)}%</span>
            {props.readOnly && <span className="bw-ro">read-only</span>}
            <span className="bw-sep" />
            <button
                type="button"
                className={`bw-btn bw-btn-ghost${props.keysVisible ? ' is-on' : ''}`}
                onClick={props.onKeys}
                aria-pressed={props.keysVisible}
                title="On-screen keyboard"
            >
                ⌨
            </button>
            <button type="button" className="bw-btn bw-btn-ghost" onClick={props.onHelp} title="Gestures and shortcuts">
                ?
            </button>
        </div>
    );
}
