import { useEffect, useRef, useState } from 'react';
import { DOUBLE_TAP_ZOOM, ZOOM_STEP, clampZoom, panBy, screenToStream, zoomAt } from '../screen/viewport';
import type { Point, ViewGeometry, Viewport } from '../screen/viewport';
import { emptyGeometry, geometryOf, resetView, settleView, transformOf } from './browserView';
import type { ViewTransform } from './browserView';

interface ZoomState {
    geometry: ViewGeometry;
    view: Viewport;
}

export interface BrowserZoom {
    stageRef: React.RefObject<HTMLDivElement | null>;
    imgRef: React.RefObject<HTMLImageElement | null>;
    geometry: ViewGeometry;
    view: Viewport;
    transform: ViewTransform;
    /** Re-reads the stage box and the decoded frame; re-clamps if either moved. */
    measure: () => ViewGeometry;
    /** Back to 1× centred, e.g. after a session or tab switch. */
    reset: () => void;
    setView: (next: Viewport) => void;
    /** Zooms to `target` around a stage point (defaults to the stage centre). */
    zoomTo: (target: number, anchor?: Point) => void;
    step: (direction: 1 | -1) => void;
    /** Double-tap behaviour: toggle between 1× and DOUBLE_TAP_ZOOM. */
    toggleAt: (anchor: Point) => void;
    pan: (dxScreen: number, dyScreen: number) => void;
}

/**
 * Zoom/pan state for the cast stage.
 *
 * Geometry lives in a ref and is mirrored into state only for rendering: a pinch
 * updates it on every pointermove, and React is fast enough for a transform but
 * not for re-reading layout in the render path.
 */
export function useBrowserZoom(): BrowserZoom {
    const stageRef = useRef<HTMLDivElement | null>(null);
    const imgRef = useRef<HTMLImageElement | null>(null);
    const store = useRef<ZoomState>({ geometry: emptyGeometry(), view: { zoom: 1, cx: 0, cy: 0 } });
    const [snap, setSnap] = useState<ZoomState>(store.current);

    const publish = (next: ZoomState) => {
        store.current = next;
        setSnap(next);
    };

    const measure = (): ViewGeometry => {
        const stage = stageRef.current;
        const img = imgRef.current;
        const current = store.current.geometry;
        if (!stage || !img || !img.naturalWidth || !img.naturalHeight) return current;
        const rect = stage.getBoundingClientRect();
        const geometry = geometryOf(rect, img.naturalWidth, img.naturalHeight);
        if (
            geometry.stageW === current.stageW &&
            geometry.stageH === current.stageH &&
            geometry.imgW === current.imgW &&
            geometry.imgH === current.imgH
        ) {
            return current;
        }
        publish({ geometry, view: settleView(geometry, store.current.view) });
        return geometry;
    };

    const reset = () => {
        const geometry = measure();
        publish({ geometry, view: resetView(geometry) });
    };

    const setView = (next: Viewport) => {
        publish({ geometry: store.current.geometry, view: settleView(store.current.geometry, next) });
    };

    const zoomTo = (target: number, anchor?: Point) => {
        const geometry = measure();
        const view = store.current.view;
        const stageCentre = { x: geometry.stageW / 2, y: geometry.stageH / 2 };
        const anchorImage = screenToStream(geometry, view, anchor ?? stageCentre);
        setView(zoomAt(geometry, view, clampZoom(target), anchorImage));
    };

    const step = (direction: 1 | -1) => {
        const view = store.current.view;
        zoomTo(direction > 0 ? view.zoom * ZOOM_STEP : view.zoom / ZOOM_STEP);
    };

    const toggleAt = (anchor: Point) => {
        const view = store.current.view;
        zoomTo(view.zoom > 1.05 ? 1 : DOUBLE_TAP_ZOOM, anchor);
    };

    const pan = (dxScreen: number, dyScreen: number) => {
        setView(panBy(store.current.geometry, store.current.view, dxScreen, dyScreen));
    };

    // The stage changes size on rotation, drawer open/close and toolbar wrap;
    // re-measuring keeps the centre of interest where the human left it.
    useEffect(() => {
        const stage = stageRef.current;
        if (!stage) return;
        const observer = new ResizeObserver(() => {
            measure();
        });
        observer.observe(stage);
        const onOrientation = () => measure();
        window.addEventListener('orientationchange', onOrientation);
        return () => {
            observer.disconnect();
            window.removeEventListener('orientationchange', onOrientation);
        };
        // measure() only touches refs and setState, so the first closure is safe.
        // eslint-disable-next-line react-hooks/exhaustive-deps
    }, []);

    return {
        stageRef,
        imgRef,
        geometry: snap.geometry,
        view: snap.view,
        transform: transformOf(snap.geometry, snap.view),
        measure,
        reset,
        setView,
        zoomTo,
        step,
        toggleAt,
        pan,
    };
}
