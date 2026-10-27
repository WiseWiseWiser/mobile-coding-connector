import { useEffect, useRef } from 'react';
import type { RefObject } from 'react';
import { BUTTON_LEFT, BUTTON_MIDDLE, BUTTON_RIGHT } from './RemoteScreen';
import type { RemoteScreen } from './RemoteScreen';
import type { ScreenMode } from './screenMode';
import { DragLatch, canStartDragHold } from './screenDrag';

interface MirrorViewProps {
    screen: RefObject<RemoteScreen | null>;
    attachCanvas: (canvas: HTMLCanvasElement | null) => void;
    active: boolean;
    mode: ScreenMode;
    onRequestControl: () => void;
    /** Fires when a long-press grabs or drops the host pointer. */
    onDragChange?: (dragging: boolean) => void;
}

const MOVE_INTERVAL_MS = 16;
const WHEEL_STEP = 40;
const TAP_SLOP_PX = 12;
const DOUBLE_TAP_MS = 320;
const DOUBLE_TAP_SLOP_PX = 30;

type TouchGesture = {
    kind: 'pan' | 'pinch';
    /** First touch position, for tap slop. */
    startX: number;
    startY: number;
    /** Previous touch position, for incremental panning. */
    lastX: number;
    lastY: number;
    moved: boolean;
    /** Image pixel held under the pinch midpoint; keeps the pinch drift free. */
    anchorX: number;
    anchorY: number;
    startDist: number;
    startZoom: number;
};

function midpoint(a: Touch, b: Touch) {
    return { x: (a.clientX + b.clientX) / 2, y: (a.clientY + b.clientY) / 2 };
}

function distance(a: Touch, b: Touch) {
    return Math.hypot(a.clientX - b.clientX, a.clientY - b.clientY) || 1;
}

/**
 * The screen surface. Which world a gesture lands in is decided by `mode`:
 *
 *  - REMOTE — mouse events map 1:1 onto host input; a tap clicks, a long press
 *    grabs the pointer and drags until the finger lifts, two fingers scroll.
 *  - WEB — the picture is panned and pinched locally and nothing is sent to the
 *    Mac, so a corner can be read without the cursor moving or a click landing.
 */
export function MirrorView({
    screen,
    attachCanvas,
    active,
    mode,
    onRequestControl,
    onDragChange,
}: MirrorViewProps) {
    const canvasRef = useRef<HTMLCanvasElement | null>(null);
    const activeRef = useRef(active);
    const modeRef = useRef(mode);
    const requestRef = useRef(onRequestControl);
    const dragNotify = useRef(onDragChange);
    const latch = useRef<DragLatch | null>(null);

    const lastMoveAt = useRef(0);
    const wheelAcc = useRef(0);
    const pressedButton = useRef(0);
    const touchScroll = useRef(0);
    const gesture = useRef<TouchGesture | null>(null);
    const lastTap = useRef<{ at: number; x: number; y: number } | null>(null);
    const mouse = useRef<{ x: number; y: number; moved: boolean } | null>(null);

    // A long press grabs the host pointer: the button goes down where the
    // hovering cursor already is, every following move drags, and the lift drops
    // it instead of clicking. Built in an effect because the latch reads refs.
    useEffect(() => {
        const created = new DragLatch(
            {
                press: () => {
                    const remote = screen.current;
                    if (!remote) return;
                    const { x, y } = remote.lastPoint;
                    remote.buttonDown(BUTTON_LEFT, x, y);
                },
                release: () => {
                    const remote = screen.current;
                    if (!remote) return;
                    const { x, y } = remote.lastPoint;
                    remote.buttonUp(BUTTON_LEFT, x, y);
                },
            },
            () => {
                const state = gesture.current;
                return (
                    !!state &&
                    state.kind === 'pan' &&
                    !state.moved &&
                    canStartDragHold(modeRef.current, activeRef.current, 1)
                );
            },
            (held) => dragNotify.current?.(held)
        );
        latch.current = created;
        return () => {
            // Never leave the button pinned because the page went away.
            created.drop(true);
            latch.current = null;
        };
    }, [screen]);

    useEffect(() => {
        dragNotify.current = onDragChange;
    }, [onDragChange]);
    useEffect(() => {
        activeRef.current = active;
        // Losing control mid-drag must not leave the button pinned on the host:
        // the server already released it with the lock, so just drop locally.
        if (!active) latch.current?.drop(false);
    }, [active]);
    useEffect(() => {
        requestRef.current = onRequestControl;
    }, [onRequestControl]);
    useEffect(() => {
        modeRef.current = mode;
        // In WEB mode the host cursor is meaningless — it is not where the user
        // is looking — so hide it and let the local picture be the whole story.
        screen.current?.setShowCursor(mode === 'remote');
        if (mode === 'web') latch.current?.drop(true);
    }, [mode, screen]);

    // Wheel and touch listeners are native so they can be non-passive: the page
    // must not scroll while the user is driving the remote desktop or panning.
    useEffect(() => {
        const canvas = canvasRef.current;
        if (!canvas) return;

        const onWheel = (event: WheelEvent) => {
            event.preventDefault();
            const remote = screen.current;
            if (!remote) return;
            if (modeRef.current === 'web') {
                remote.zoomByClient(event.deltaY < 0 ? 1 : -1, event.clientX, event.clientY);
                return;
            }
            const point = remote.toImageCoords(event);
            if (!point) return;
            if (!activeRef.current) {
                requestRef.current();
                return;
            }
            wheelAcc.current += event.deltaY;
            if (Math.abs(wheelAcc.current) < WHEEL_STEP) return;
            const lines = Math.sign(wheelAcc.current);
            wheelAcc.current = 0;
            remote.wheel(0, lines, point.x, point.y);
        };

        const startPinch = (touches: TouchList) => {
            const remote = screen.current;
            const a = touches[0];
            const b = touches[1];
            if (!remote || !a || !b) return;
            const mid = midpoint(a, b);
            const anchor = remote.streamAt(mid.x, mid.y);
            gesture.current = {
                kind: 'pinch',
                startX: mid.x,
                startY: mid.y,
                lastX: mid.x,
                lastY: mid.y,
                moved: false,
                anchorX: anchor?.x ?? 0,
                anchorY: anchor?.y ?? 0,
                startDist: distance(a, b),
                startZoom: remote.zoom,
            };
        };

        const startPan = (touch: Touch) => {
            gesture.current = {
                kind: 'pan',
                startX: touch.clientX,
                startY: touch.clientY,
                lastX: touch.clientX,
                lastY: touch.clientY,
                moved: false,
                anchorX: 0,
                anchorY: 0,
                startDist: 1,
                startZoom: 1,
            };
        };

        const onTouchStart = (event: TouchEvent) => {
            const remote = screen.current;
            if (!remote) return;

            if (modeRef.current === 'web') {
                if (event.touches.length >= 2) {
                    startPinch(event.touches);
                    return;
                }
                const t = event.touches[0];
                if (t) startPan(t);
                return;
            }

            if (event.touches.length === 1 && !activeRef.current) {
                requestRef.current();
                return;
            }
            if (event.touches.length === 2) {
                // A second finger scrolls instead of dragging; give the button
                // back before the wheel starts or it would stay held down.
                latch.current?.drop(true);
                gesture.current = null;
                touchScroll.current = (event.touches[0].clientY + event.touches[1].clientY) / 2;
                return;
            }
            const t = event.touches[0];
            if (!t) return;
            startPan(t);
            lastTap.current = null;
            const point = remote.toImageCoords(t);
            if (point) remote.move(point.x, point.y);
            latch.current?.arm();
        };

        const onTouchMove = (event: TouchEvent) => {
            const remote = screen.current;
            if (!remote) return;

            if (modeRef.current === 'web') {
                const state = gesture.current;
                if (!state) return;
                event.preventDefault();
                if (state.kind === 'pinch') {
                    const a = event.touches[0];
                    const b = event.touches[1];
                    if (!a || !b) return;
                    const mid = midpoint(a, b);
                    const zoom = state.startZoom * (distance(a, b) / state.startDist);
                    // Anchor on the image pixel grabbed at the midpoint so the
                    // picture follows the fingers instead of sliding away.
                    remote.placeStreamAt(state.anchorX, state.anchorY, mid.x, mid.y, zoom);
                    return;
                }
                const t = event.touches[0];
                if (!t) return;
                const dx = t.clientX - state.lastX;
                const dy = t.clientY - state.lastY;
                state.lastX = t.clientX;
                state.lastY = t.clientY;
                if (
                    Math.abs(t.clientX - state.startX) > TAP_SLOP_PX ||
                    Math.abs(t.clientY - state.startY) > TAP_SLOP_PX
                ) {
                    state.moved = true;
                }
                remote.panByScreen(dx, dy);
                return;
            }

            if (event.touches.length === 2) {
                event.preventDefault();
                const y = (event.touches[0].clientY + event.touches[1].clientY) / 2;
                const dy = touchScroll.current - y;
                touchScroll.current = y;
                if (Math.abs(dy) < 6) return;
                const point = remote.lastPoint;
                remote.wheel(0, dy > 0 ? -1 : 1, point.x, point.y);
                return;
            }
            const t = event.touches[0];
            const state = gesture.current;
            if (!t || !state) return;
            event.preventDefault();
            if (
                Math.abs(t.clientX - state.startX) > TAP_SLOP_PX ||
                Math.abs(t.clientY - state.startY) > TAP_SLOP_PX
            ) {
                state.moved = true;
            }
            // Once the finger travels, the press is a pan/hover, not a grab.
            if (state.moved) latch.current?.cancel();
            const point = remote.toImageCoords(t);
            if (point) remote.move(point.x, point.y);
        };

        const onTouchEnd = (event: TouchEvent) => {
            const remote = screen.current;
            if (!remote) return;
            const state = gesture.current;

            if (modeRef.current === 'web') {
                // One finger lifted from a pinch: keep panning from where the
                // remaining finger is, without jumping.
                const rest = event.touches[0];
                if (rest && event.touches.length === 1) {
                    startPan(rest);
                    gesture.current!.moved = true;
                    return;
                }
                gesture.current = null;
                // A pinch is never a tap, even if the fingers never travelled.
                if (!state || state.moved || state.kind !== 'pan') {
                    lastTap.current = null;
                    return;
                }
                const now = Date.now();
                const previous = lastTap.current;
                const isDouble =
                    !!previous &&
                    now - previous.at < DOUBLE_TAP_MS &&
                    Math.abs(previous.x - state.startX) < DOUBLE_TAP_SLOP_PX &&
                    Math.abs(previous.y - state.startY) < DOUBLE_TAP_SLOP_PX;
                if (isDouble) {
                    lastTap.current = null;
                    remote.zoomToggleClient(state.startX, state.startY);
                } else {
                    lastTap.current = { at: now, x: state.startX, y: state.startY };
                }
                return;
            }

            gesture.current = null;
            latch.current?.cancel();
            if (latch.current?.active) {
                latch.current.drop(true);
                return;
            }
            if (!state || state.moved || !activeRef.current) return;
            remote.clickAt(BUTTON_LEFT);
        };

        // A cancelled touch (callout, system gesture, interrupted call) is never
        // a click, but it must still drop the button if one was grabbed.
        const onTouchCancel = () => {
            gesture.current = null;
            touchScroll.current = 0;
            latch.current?.drop(true);
        };

        canvas.addEventListener('wheel', onWheel, { passive: false });
        canvas.addEventListener('touchstart', onTouchStart, { passive: true });
        canvas.addEventListener('touchmove', onTouchMove, { passive: false });
        canvas.addEventListener('touchend', onTouchEnd);
        canvas.addEventListener('touchcancel', onTouchCancel);
        return () => {
            canvas.removeEventListener('wheel', onWheel);
            canvas.removeEventListener('touchstart', onTouchStart);
            canvas.removeEventListener('touchmove', onTouchMove);
            canvas.removeEventListener('touchend', onTouchEnd);
            canvas.removeEventListener('touchcancel', onTouchCancel);
        };
    }, [screen]);

    const onPointerDown = (event: React.PointerEvent<HTMLCanvasElement>) => {
        if (event.pointerType !== 'mouse') return;
        const remote = screen.current;
        if (!remote) return;

        if (mode === 'web') {
            event.currentTarget.setPointerCapture(event.pointerId);
            mouse.current = { x: event.clientX, y: event.clientY, moved: false };
            pressedButton.current = 0;
            return;
        }

        if (!active) {
            onRequestControl();
            return;
        }
        const point = remote.toImageCoords(event);
        if (!point) return;
        event.currentTarget.setPointerCapture(event.pointerId);
        const button = event.button === 2 ? BUTTON_RIGHT : event.button === 1 ? BUTTON_MIDDLE : BUTTON_LEFT;
        pressedButton.current = button;
        remote.buttonDown(button, point.x, point.y);
    };

    const onPointerMove = (event: React.PointerEvent<HTMLCanvasElement>) => {
        if (event.pointerType !== 'mouse') return;
        const remote = screen.current;
        if (!remote) return;

        if (mode === 'web') {
            const state = mouse.current;
            if (!state) return;
            const dx = event.clientX - state.x;
            const dy = event.clientY - state.y;
            state.x = event.clientX;
            state.y = event.clientY;
            if (Math.abs(dx) > 0 || Math.abs(dy) > 0) {
                state.moved = true;
                remote.panByScreen(dx, dy);
            }
            return;
        }

        if (!active) return;
        const now = performance.now();
        if (now - lastMoveAt.current < MOVE_INTERVAL_MS) return;
        lastMoveAt.current = now;
        const point = remote.toImageCoords(event);
        if (point) remote.move(point.x, point.y);
    };

    const onPointerUp = (event: React.PointerEvent<HTMLCanvasElement>) => {
        if (event.pointerType !== 'mouse') return;
        const remote = screen.current;
        if (!remote) return;

        if (mode === 'web') {
            const state = mouse.current;
            mouse.current = null;
            if (!state || state.moved) {
                lastTap.current = null;
                return;
            }
            const now = Date.now();
            const previous = lastTap.current;
            const isDouble =
                !!previous &&
                now - previous.at < DOUBLE_TAP_MS &&
                Math.abs(previous.x - event.clientX) < DOUBLE_TAP_SLOP_PX &&
                Math.abs(previous.y - event.clientY) < DOUBLE_TAP_SLOP_PX;
            if (isDouble) {
                lastTap.current = null;
                remote.zoomToggleClient(event.clientX, event.clientY);
            } else {
                lastTap.current = { at: now, x: event.clientX, y: event.clientY };
            }
            return;
        }

        if (!active) return;
        const point = remote.toImageCoords(event);
        if (!point) return;
        remote.buttonUp(pressedButton.current || BUTTON_LEFT, point.x, point.y);
        pressedButton.current = 0;
    };

    return (
        <canvas
            ref={(el) => {
                canvasRef.current = el;
                attachCanvas(el);
            }}
            className={mode === 'web' ? 'screen-canvas screen-canvas-browse' : 'screen-canvas'}
            onPointerDown={onPointerDown}
            onPointerMove={onPointerMove}
            onPointerUp={onPointerUp}
            onPointerLeave={onPointerUp}
            onContextMenu={(event) => event.preventDefault()}
        />
    );
}
