import { clampZoom, panBy, screenToStream, zoomAt } from '../screen/viewport';
import type { Point, ViewGeometry, Viewport } from '../screen/viewport';

/** A press shorter than this that did not move is a tap. */
const TAP_MS = 500;
/** Two taps closer together than this toggle the zoom. */
const DOUBLE_TAP_MS = 340;
const DOUBLE_TAP_SLOP = 28;
/** Movement past this many CSS pixels means the human is dragging, not tapping. */
const TAP_SLOP = 10;

export interface ZoomPadHost {
    geometry(): ViewGeometry;
    view(): Viewport;
    setView(next: Viewport): void;
    lens(): 'view' | 'ctrl';
    /** A second finger landed (true) or the last one left (false). */
    onMultiTouch(active: boolean): void;
    /** Double tap in VIEW mode, in stage-local pixels. */
    onDoubleTap(anchor: Point): void;
}

function distance(a: Point, b: Point): number {
    return Math.hypot(a.x - b.x, a.y - b.y);
}

function midpoint(a: Point, b: Point): Point {
    return { x: (a.x + b.x) / 2, y: (a.y + b.y) / 2 };
}

/**
 * Pan, pinch and double-tap for the cast stage.
 *
 * Two fingers always mean the viewer: pinch scales about the midpoint while the
 * midpoint itself pans, so the page stays glued to the fingers. One finger means
 * the viewer only in VIEW mode; in CTRL the same finger belongs to the page and
 * this class stays out of the way, except that a second finger still zooms.
 */
export class BrowserZoomPad {
    private stage: HTMLElement | null = null;
    private host: ZoomPadHost;
    private points = new Map<number, Point>();
    private pinch: { d: number; m: Point; view: Viewport } | null = null;
    private pan: { p: Point; view: Viewport } | null = null;
    private downAt = 0;
    private downPoint: Point | null = null;
    private moved = false;
    private pinched = false;
    private lastTap = { at: 0, x: 0, y: 0 };

    constructor(host: ZoomPadHost) {
        this.host = host;
    }

    attach(stage: HTMLElement) {
        this.detach();
        this.stage = stage;
        stage.addEventListener('pointerdown', this.onPointerDown);
        stage.addEventListener('pointermove', this.onPointerMove);
        stage.addEventListener('pointerup', this.onPointerUp);
        stage.addEventListener('pointercancel', this.onPointerCancel);
    }

    detach() {
        const stage = this.stage;
        if (!stage) return;
        stage.removeEventListener('pointerdown', this.onPointerDown);
        stage.removeEventListener('pointermove', this.onPointerMove);
        stage.removeEventListener('pointerup', this.onPointerUp);
        stage.removeEventListener('pointercancel', this.onPointerCancel);
        this.stage = null;
        this.points.clear();
        this.pinch = null;
        this.pan = null;
        this.downPoint = null;
    }

    private local(event: PointerEvent): Point {
        const rect = this.stage?.getBoundingClientRect();
        if (!rect) return { x: 0, y: 0 };
        return { x: event.clientX - rect.left, y: event.clientY - rect.top };
    }

    private onPointerDown = (event: PointerEvent) => {
        if (!this.stage) return;
        if (event.pointerType === 'mouse' && event.button !== 0) return;
        if ((event.target as HTMLElement | null)?.closest('button')) return;
        const point = this.local(event);
        this.points.set(event.pointerId, point);
        try {
            this.stage.setPointerCapture?.(event.pointerId);
        } catch {
            /* capture is a nicety, not a requirement */
        }
        if (this.points.size === 1) {
            this.downAt = Date.now();
            this.downPoint = point;
            this.moved = false;
            this.pinched = false;
            this.pan = this.host.lens() === 'view' ? { p: point, view: this.host.view() } : null;
            return;
        }
        if (this.points.size === 2) {
            const [a, b] = [...this.points.values()];
            this.pan = null;
            this.pinched = true;
            this.pinch = { d: distance(a, b), m: midpoint(a, b), view: this.host.view() };
            this.host.onMultiTouch(true);
            return;
        }
        // A third finger has no meaning; stop tracking the pinch rather than
        // jumping when one of them lifts.
        this.pinch = null;
    };

    private onPointerMove = (event: PointerEvent) => {
        if (!this.points.has(event.pointerId)) return;
        const point = this.local(event);
        this.points.set(event.pointerId, point);
        if (this.downPoint && distance(point, this.downPoint) > TAP_SLOP) this.moved = true;

        const geometry = this.host.geometry();
        if (this.pinch && this.points.size >= 2) {
            const [a, b] = [...this.points.values()];
            const spread = distance(a, b);
            if (!this.pinch.d || !spread) return;
            const mid = midpoint(a, b);
            const anchor = screenToStream(geometry, this.pinch.view, this.pinch.m);
            const target = clampZoom(this.pinch.view.zoom * (spread / this.pinch.d));
            const zoomed = zoomAt(geometry, this.pinch.view, target, anchor);
            this.host.setView(panBy(geometry, zoomed, mid.x - this.pinch.m.x, mid.y - this.pinch.m.y));
            return;
        }
        if (!this.pan) return;
        this.host.setView(panBy(geometry, this.pan.view, point.x - this.pan.p.x, point.y - this.pan.p.y));
    };

    private onPointerUp = (event: PointerEvent) => {
        this.release(event);
    };

    private onPointerCancel = (event: PointerEvent) => {
        this.release(event);
    };

    private release(event: PointerEvent) {
        if (!this.points.has(event.pointerId)) return;
        const point = this.local(event);
        this.points.delete(event.pointerId);
        try {
            this.stage?.releasePointerCapture?.(event.pointerId);
        } catch {
            /* already released */
        }
        if (this.points.size < 2) this.pinch = null;
        if (this.points.size > 0) return;
        this.pan = null;
        this.host.onMultiTouch(false);
        if (event.type === 'pointercancel') return;
        if (this.host.lens() !== 'view') return;
        if (this.moved || this.pinched) return;
        if (Date.now() - this.downAt > TAP_MS) return;
        const now = Date.now();
        const isDouble =
            now - this.lastTap.at < DOUBLE_TAP_MS &&
            Math.abs(point.x - this.lastTap.x) < DOUBLE_TAP_SLOP &&
            Math.abs(point.y - this.lastTap.y) < DOUBLE_TAP_SLOP;
        if (isDouble) {
            this.lastTap = { at: 0, x: 0, y: 0 };
            this.host.onDoubleTap(point);
            return;
        }
        this.lastTap = { at: now, x: point.x, y: point.y };
    }
}
