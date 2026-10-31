/* eslint-disable react-hooks/refs -- The gesture controllers are imperative and must read the latest props without being re-created; these refs are only read from pointer handlers, never during render. */
import { useEffect, useRef, useState } from 'react';
import { BrowserGestures } from './browserGestures';
import type { GestureHost } from './browserGestures';
import { BrowserZoomPad } from './browserZoomPad';
import type { BrowserZoom } from './useBrowserZoom';
import type { ClientMessage } from './browserWire';

export interface BrowserGestureOptions {
    zoom: BrowserZoom;
    lens: 'view' | 'ctrl';
    send: (message: ClientMessage) => void;
    attachStage: (el: HTMLDivElement | null) => void;
    attachImage: (el: HTMLImageElement | null) => void;
    setInput: (active: boolean) => void;
    setDragging: (dragging: boolean) => void;
}

/** Gesture and pinch-zoom wiring shared by the desktop and phone stages. */
export function useBrowserGestures(options: BrowserGestureOptions) {
    const { zoom, lens, send, attachStage, attachImage, setInput, setDragging } = options;
    const stageEl = useRef<HTMLDivElement | null>(null);
    const sendRef = useRef(send);
    const lensRef = useRef(lens);
    const zoomRef = useRef(zoom);

    useEffect(() => {
        sendRef.current = send;
        lensRef.current = lens;
        zoomRef.current = zoom;
    }, [send, lens, zoom]);

    const [gestures] = useState(
        () =>
            new BrowserGestures({
                view: () => ({ geometry: zoomRef.current.geometry, view: zoomRef.current.view }),
                tap: (x, y) => sendRef.current({ type: 'tap', x, y, button: 1 }),
                down: (x, y) => sendRef.current({ type: 'down', x, y, button: 1 }),
                move: (x, y) => sendRef.current({ type: 'move', x, y, button: 1 }),
                up: (x, y) => sendRef.current({ type: 'up', x, y, button: 1 }),
                wheel: (x, y, dx, dy) =>
                    sendRef.current({ type: 'wheel', x, y, delta_x: dx, delta_y: dy }),
                onDragChange: setDragging,
            } satisfies GestureHost),
    );
    const [zoomPad] = useState(
        () =>
            new BrowserZoomPad({
                geometry: () => zoomRef.current.geometry,
                view: () => zoomRef.current.view,
                setView: (next) => zoomRef.current.setView(next),
                lens: () => lensRef.current,
                onMultiTouch: (active) => {
                    // A second finger means zoom, not a click: cancel whatever the
                    // first finger had started before the pad takes over.
                    if (active) {
                        gestures.standDown();
                        setDragging(false);
                    }
                },
                onDoubleTap: (anchor) => zoomRef.current.toggleAt(anchor),
            }),
    );

    // One stable ref callback: a new closure every render would detach and
    // reattach the gesture listeners on every frame. The stage itself is
    // replaced when the layout crosses 900 px, so the pad follows the ref
    // instead of assuming the first element lives forever.
    const [stageRefCb] = useState(() => (el: HTMLDivElement | null) => {
        stageEl.current = el;
        zoom.stageRef.current = el;
        if (el) zoomPad.attach(el);
        else zoomPad.detach();
        attachStage(el);
    });
    const [imgRefCb] = useState(() => (el: HTMLImageElement | null) => {
        zoom.imgRef.current = el;
        attachImage(el);
    });

    const [inputSink] = useState(() => setInput);

    useEffect(() => {
        inputSink(lens === 'ctrl');
        return () => inputSink(false);
    }, [lens, inputSink]);

    return { stageRefCb, imgRefCb };
}
