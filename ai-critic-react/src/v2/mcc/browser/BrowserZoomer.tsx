export interface BrowserZoomerProps {
    canZoom: boolean;
    onZoomIn: () => void;
    onZoomOut: () => void;
    onFit: () => void;
}

/** Floating −/+/Fit controls over the cast, sized for both mouse and thumb. */
export function BrowserZoomer(props: BrowserZoomerProps) {
    return (
        <div className="bw-zoomer" role="group" aria-label="Zoom">
            <button
                type="button"
                className="bw-zbtn"
                data-z="out"
                onClick={props.onZoomOut}
                disabled={!props.canZoom}
                title="Zoom out"
                aria-label="Zoom out"
            >
                −
            </button>
            <button
                type="button"
                className="bw-zbtn"
                data-z="in"
                onClick={props.onZoomIn}
                disabled={!props.canZoom}
                title="Zoom in"
                aria-label="Zoom in"
            >
                +
            </button>
            <button type="button" className="bw-zbtn bw-zfit" data-z="fit" onClick={props.onFit} title="Fit to stage">
                Fit
            </button>
        </div>
    );
}
