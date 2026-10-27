import type { ViewGeometry, Viewport } from '../screen/viewport';
import { hasGeometry, localPoint, normalizeAt } from './browserView';

/** How long a finger must rest before it grabs the page. */
export const DRAG_HOLD_MS = 350;
/** Movement past this many CSS pixels turns a press into a scroll. */
export const SCROLL_SLOP = 8;
const MOVE_INTERVAL_MS = 33;
const WHEEL_INTERVAL_MS = 30;

export interface LiveView {
    geometry: ViewGeometry;
    view: Viewport;
}

export interface GestureHost {
    /** Current zoom/pan, so coordinates survive the transformed frame. */
    view?: () => LiveView;
    tap(x: number, y: number): void;
    down(x: number, y: number): void;
    move(x: number, y: number): void;
    up(x: number, y: number): void;
    wheel(x: number, y: number, dx: number, dy: number): void;
    onDragChange?: (dragging: boolean) => void;
}

interface Point {
    x: number;
    y: number;
}

/** Where the image actually paints inside its element, letterboxing removed. */
export function contentBox(img: HTMLImageElement | null, fallback: HTMLElement): { left: number; top: number; width: number; height: number } {
    const rect = (img ?? fallback).getBoundingClientRect();
    const naturalWidth = img?.naturalWidth ?? 0;
    const naturalHeight = img?.naturalHeight ?? 0;
    if (!naturalWidth || !naturalHeight) {
        return { left: rect.left, top: rect.top, width: rect.width, height: rect.height };
    }
    const scale = Math.min(rect.width / naturalWidth, rect.height / naturalHeight);
    const width = naturalWidth * scale;
    const height = naturalHeight * scale;
    return {
        left: rect.left + (rect.width - width) / 2,
        top: rect.top + (rect.height - height) / 2,
        width,
        height,
    };
}

/**
 * Translates one finger on the picture into browser input.
 *
 * A phone has no mouse, so three gestures have to share the glass:
 *  - a quick tap clicks;
 *  - a moving finger scrolls the page (wheel events, so it works over any
 *    scrollable container, not just the document);
 *  - a finger that rests for DRAG_HOLD_MS grabs the pointer and drags, for
 *    carousels, sliders and text selection.
 *
 * Coordinates arrive normalized so the client does not care which viewport the
 * window achieved.
 */
export class BrowserGestures {
    private container: HTMLElement | null = null;
    private img: HTMLImageElement | null = null;
    private cursor: HTMLElement | null = null;
    private host: GestureHost;
    private holdTimer: ReturnType<typeof setTimeout> | null = null;
    private wheelTimer: ReturnType<typeof setTimeout> | null = null;
    private moveTimer: ReturnType<typeof setTimeout> | null = null;

    private pointerId: number | null = null;
    private start: Point = { x: 0, y: 0 };
    private last: Point = { x: 0, y: 0 };
    private latched = false;
    private scrolled = false;
    private pendingMove: Point | null = null;
    private lastMoveSentAt = 0;
    private wheelDx = 0;
    private wheelDy = 0;
    private wheelAt: Point = { x: 0, y: 0 };

    constructor(host: GestureHost) {
        this.host = host;
    }

    attach(container: HTMLElement | null, img: HTMLImageElement | null) {
        this.detach();
        if (!container) return;
        this.container = container;
        this.img = img;
        container.addEventListener('pointerdown', this.onPointerDown);
        container.addEventListener('pointermove', this.onPointerMove);
        container.addEventListener('pointerup', this.onPointerUp);
        container.addEventListener('pointercancel', this.onPointerCancel);
        container.addEventListener('wheel', this.onWheel, { passive: false });
    }

    detach() {
        const container = this.container;
        if (container) {
            container.removeEventListener('pointerdown', this.onPointerDown);
            container.removeEventListener('pointermove', this.onPointerMove);
            container.removeEventListener('pointerup', this.onPointerUp);
            container.removeEventListener('pointercancel', this.onPointerCancel);
            container.removeEventListener('wheel', this.onWheel);
        }
        this.clearHold();
        if (this.wheelTimer) clearTimeout(this.wheelTimer);
        if (this.moveTimer) clearTimeout(this.moveTimer);
        this.wheelTimer = null;
        this.moveTimer = null;
        this.container = null;
    }

    attachCursor(cursor: HTMLElement | null) {
        this.cursor = cursor;
    }

    private normalize(clientX: number, clientY: number): Point {
        const container = this.container;
        if (!container) return { x: 0, y: 0 };
        const live = this.host.view?.();
        if (live && hasGeometry(live.geometry)) {
            const rect = container.getBoundingClientRect();
            return normalizeAt(live.geometry, live.view, localPoint(rect, clientX, clientY));
        }
        const box = contentBox(this.img, container);
        if (box.width <= 0 || box.height <= 0) return { x: 0, y: 0 };
        return {
            x: Math.min(1, Math.max(0, (clientX - box.left) / box.width)),
            y: Math.min(1, Math.max(0, (clientY - box.top) / box.height)),
        };
    }

    /**
     * Abandons the current finger without sending anything.
     *
     * A second finger means the human is zooming, not clicking: the hold timer
     * and any half-scrolled wheel delta must not leak into the page.
     */
    standDown() {
        this.clearHold();
        this.latched = false;
        this.scrolled = false;
        this.pendingMove = null;
        if (this.moveTimer) {
            clearTimeout(this.moveTimer);
            this.moveTimer = null;
        }
        if (this.wheelTimer) {
            clearTimeout(this.wheelTimer);
            this.wheelTimer = null;
        }
        this.wheelDx = 0;
        this.wheelDy = 0;
        const pointerId = this.pointerId;
        this.pointerId = null;
        if (this.container && pointerId !== null) {
            try {
                this.container.releasePointerCapture?.(pointerId);
            } catch {
                /* capture already gone */
            }
        }
        this.host.onDragChange?.(false);
    }

    private moveCursor(clientX: number, clientY: number) {
        const container = this.container;
        const cursor = this.cursor;
        if (!container || !cursor) return;
        const rect = container.getBoundingClientRect();
        cursor.style.transform = `translate(${clientX - rect.left}px, ${clientY - rect.top}px)`;
        cursor.style.opacity = '1';
    }

    private clearHold() {
        if (this.holdTimer) {
            clearTimeout(this.holdTimer);
            this.holdTimer = null;
        }
    }

    private onPointerDown = (event: PointerEvent) => {
        if (!event.isPrimary || this.pointerId !== null) return;
        if (event.pointerType === 'mouse' && event.button !== 0) return;
        // Overlay controls (zoom buttons) sit inside the stage; leave them alone.
        if ((event.target as HTMLElement | null)?.closest('button')) return;
        this.pointerId = event.pointerId;
        this.start = { x: event.clientX, y: event.clientY };
        this.last = this.start;
        this.latched = false;
        this.scrolled = false;
        this.container?.setPointerCapture?.(event.pointerId);
        this.moveCursor(event.clientX, event.clientY);
        this.clearHold();
        this.holdTimer = setTimeout(() => {
            this.holdTimer = null;
            if (this.pointerId === null || this.scrolled) return;
            this.latched = true;
            const point = this.normalize(this.last.x, this.last.y);
            this.host.down(point.x, point.y);
            this.host.onDragChange?.(true);
        }, DRAG_HOLD_MS);
    };

    private onPointerMove = (event: PointerEvent) => {
        if (event.pointerId !== this.pointerId) return;
        const previous = this.last;
        this.last = { x: event.clientX, y: event.clientY };
        this.moveCursor(event.clientX, event.clientY);
        if (this.latched) {
            this.queueMove();
            return;
        }
        const distance = Math.hypot(event.clientX - this.start.x, event.clientY - this.start.y);
        if (distance <= SCROLL_SLOP) return;
        this.scrolled = true;
        this.clearHold();
        this.queueWheel(event.clientX, event.clientY, event.clientX - previous.x, event.clientY - previous.y);
    };

    private onPointerUp = (event: PointerEvent) => {
        if (event.pointerId !== this.pointerId) return;
        this.finish(event.clientX, event.clientY, false);
    };

    private onPointerCancel = (event: PointerEvent) => {
        if (event.pointerId !== this.pointerId) return;
        this.finish(event.clientX, event.clientY, true);
    };

    private finish(clientX: number, clientY: number, cancelled: boolean) {
        this.clearHold();
        const point = this.normalize(clientX, clientY);
        const wasLatched = this.latched;
        const pointerId = this.pointerId;
        this.latched = false;
        this.pointerId = null;
        if (this.container && pointerId !== null) {
            try {
                this.container.releasePointerCapture?.(pointerId);
            } catch {
                /* capture already gone */
            }
        }
        if (wasLatched) {
            this.host.up(point.x, point.y);
            this.host.onDragChange?.(false);
            return;
        }
        if (this.scrolled) {
            this.flushWheel();
            return;
        }
        if (!cancelled) this.host.tap(point.x, point.y);
    }

    private queueMove() {
        this.pendingMove = this.last;
        if (this.moveTimer) return;
        const wait = Math.max(0, MOVE_INTERVAL_MS - (Date.now() - this.lastMoveSentAt));
        this.moveTimer = setTimeout(() => {
            this.moveTimer = null;
            const point = this.pendingMove;
            this.pendingMove = null;
            this.lastMoveSentAt = Date.now();
            if (!point) return;
            const normalized = this.normalize(point.x, point.y);
            this.host.move(normalized.x, normalized.y);
        }, wait);
    }

    private queueWheel(clientX: number, clientY: number, dx: number, dy: number) {
        // Finger travelling down means the page should travel up with it.
        this.wheelDx -= dx;
        this.wheelDy -= dy;
        this.wheelAt = this.normalize(clientX, clientY);
        if (this.wheelTimer) return;
        this.wheelTimer = setTimeout(() => {
            this.wheelTimer = null;
            this.flushWheel();
        }, WHEEL_INTERVAL_MS);
    }

    private flushWheel() {
        if (this.wheelTimer) {
            clearTimeout(this.wheelTimer);
            this.wheelTimer = null;
        }
        if (!this.wheelDx && !this.wheelDy) return;
        this.host.wheel(this.wheelAt.x, this.wheelAt.y, this.wheelDx, this.wheelDy);
        this.wheelDx = 0;
        this.wheelDy = 0;
    }

    private onWheel = (event: WheelEvent) => {
        const point = this.normalize(event.clientX, event.clientY);
        const scale = event.deltaMode === 1 ? 16 : event.deltaMode === 2 ? 400 : 1;
        this.host.wheel(point.x, point.y, event.deltaX * scale, event.deltaY * scale);
    };
}
