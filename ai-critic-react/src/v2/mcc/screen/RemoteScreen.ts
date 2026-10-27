import type { ScreenStatus } from '../../../api/screen';
import {
    centreView,
    clampView,
    drawRectOf,
    panBy,
    scaleOf,
    screenToImage,
    screenToStream,
    stepZoom,
    streamToScreen,
    toggleZoom,
    zoomAt,
} from './viewport';
import type { Point, ViewGeometry, Viewport } from './viewport';

const MSG_FRAME = 0x01;
const MSG_CURSOR = 0x02;
const MSG_POINTER = 0x81;
const MSG_KEY = 0x82;
const MSG_TEXT = 0x83;
const MSG_RELEASE_ALL = 0x84;

export const POINTER_MOVE = 1;
export const POINTER_DOWN = 2;
export const POINTER_UP = 3;
export const POINTER_WHEEL = 4;

export const BUTTON_LEFT = 1;
export const BUTTON_MIDDLE = 2;
export const BUTTON_RIGHT = 3;

const FRAME_HEADER = 22;
const TILE_HEADER = 8;
const RECONNECT_MS = 1200;

// The host pointer is polled at 30 Hz, so a live socket always carries traffic.
// Silence therefore means the connection is half-open — iOS and Cloudflare both
// keep the TCP session (and even the close event) while swallowing everything
// the host sends. Nothing recovers from that on its own: the canvas simply
// freezes on the last frame it painted.
const WATCHDOG_MS = 15000;
const WATCHDOG_TICK_MS = 3000;

export type ControlState = 'view' | 'active' | 'locked';
export type ConnectionState = 'connecting' | 'open' | 'closed';

interface CursorState {
    x: number;
    y: number;
    shape: number;
    buttons: number;
    visible: boolean;
}

export interface RemoteScreenHandlers {
    onStatus?: (status: ScreenStatus) => void;
    onControl?: (state: ControlState) => void;
    onConnection?: (state: ConnectionState) => void;
    /** `code` is the server's machine reason when it sent one; a release
     *  broadcast reports `released` so callers can tell the two apart. */
    onError?: (message: string, code?: string) => void;
    onView?: (view: Viewport) => void;
}

/**
 * Owns the screen WebSocket, the offscreen frame buffer and the pointer drawing.
 *
 * Two rules keep this simple:
 *  - frames are applied strictly in order (tiles are incremental, so a dropped
 *    frame would leave holes in the picture forever);
 *  - the visible canvas is only touched inside requestAnimationFrame.
 */
export class RemoteScreen {
    private ws: WebSocket | null = null;
    private disposed = false;
    private reconnectTimer: number | null = null;
    private watchdogTimer: number | null = null;
    private lastMessageAt = 0;
    private label: string;

    private buffer: HTMLCanvasElement | null = null;
    private bufferCtx: CanvasRenderingContext2D | null = null;
    private display: HTMLCanvasElement | null = null;
    private displayCtx: CanvasRenderingContext2D | null = null;
    private rafPending = false;
    private queue: Promise<void> = Promise.resolve();

    private imgW = 0;
    private imgH = 0;
    private cursor: CursorState = { x: 0, y: 0, shape: 0, buttons: 0, visible: false };
    private handlers: RemoteScreenHandlers;

    status: ScreenStatus | null = null;
    control: ControlState = 'view';
    connection: ConnectionState = 'closed';
    viewerId = '';
    /** True while the page is hidden: the socket is closed on purpose. */
    paused = false;
    // releasing marks a release this viewer requested, so its own broadcast does
    // not raise an error toast.
    releasing = false;

    constructor(label: string, handlers: RemoteScreenHandlers = {}) {
        this.label = label;
        this.handlers = handlers;
    }

    // ------------------------------------------------------------ connection

    connect() {
        if (this.disposed || this.paused) return;
        this.clearReconnect();
        const protocol = window.location.protocol === 'https:' ? 'wss:' : 'ws:';
        const url = `${protocol}//${window.location.host}/api/screen/ws?label=${encodeURIComponent(this.label)}`;
        this.setConnection('connecting');
        const ws = new WebSocket(url);
        ws.binaryType = 'arraybuffer';
        this.ws = ws;
        this.lastMessageAt = Date.now();
        this.startWatchdog();

        ws.onopen = () => {
            if (this.ws !== ws) return;
            this.setConnection('open');
        };
        ws.onclose = () => {
            if (this.ws !== ws) return;
            this.ws = null;
            this.setConnection('closed');
            this.setControl('view');
            this.scheduleReconnect();
        };
        ws.onerror = () => {
            this.handlers.onError?.('screen connection failed');
        };
        ws.onmessage = (event) => {
            if (this.ws !== ws) return;
            this.lastMessageAt = Date.now();
            if (typeof event.data === 'string') {
                this.handleText(event.data);
            } else {
                this.handleBinary(event.data as ArrayBuffer);
            }
        };
    }

    /**
     * Drops a socket that has gone quiet. resume() is the right recovery: a new
     * socket also earns a fresh keyframe, so the canvas repaints from scratch.
     */
    private startWatchdog() {
        if (this.watchdogTimer !== null || this.disposed) return;
        this.watchdogTimer = window.setInterval(() => {
            if (this.disposed || !this.ws) return;
            if (Date.now() - this.lastMessageAt < WATCHDOG_MS) return;
            this.handlers.onError?.('screen connection stalled, reconnecting');
            this.reconnectNow();
        }, WATCHDOG_TICK_MS);
    }

    private stopWatchdog() {
        if (this.watchdogTimer !== null) {
            window.clearInterval(this.watchdogTimer);
            this.watchdogTimer = null;
        }
    }

    /**
     * Stops watching. A hidden page is not a client: the server starts capture
     * for the first viewer and stops it when the last one disconnects, so
     * keeping the socket open made a backgrounded tab (or a sleeping phone)
     * record the host screen and hold the display awake with nobody looking.
     * The last frame stays on the canvas for an instant repaint on return, and
     * the server releases this viewer's control lock as the socket closes.
     */
    pause() {
        if (this.disposed || this.paused) return;
        this.paused = true;
        this.clearReconnect();
        this.stopWatchdog();
        if (this.ws) {
            const ws = this.ws;
            this.ws = null;
            try {
                ws.close();
            } catch {
                // already closing
            }
        }
        this.setConnection('closed');
        this.setControl('view');
    }

    /**
     * Back in front: reconnect from scratch so the canvas repaints from a fresh
     * keyframe instead of resuming on a stale picture. A socket that went quiet
     * without closing is recovered the same way by the watchdog.
     */
    resume() {
        if (this.disposed) return;
        const wasPaused = this.paused;
        this.paused = false;
        // Duplicate signals (visibilitychange and pageshow both fire on return)
        // must not churn a healthy socket.
        if (!wasPaused && this.ws) return;
        this.reconnectNow();
    }

    private reconnectNow() {
        if (this.disposed) return;
        if (this.ws) {
            const ws = this.ws;
            this.ws = null;
            try {
                ws.close();
            } catch {
                // already closing
            }
        }
        this.connect();
    }

    dispose() {
        this.disposed = true;
        this.clearReconnect();
        this.stopWatchdog();
        if (this.ws) {
            try {
                this.ws.close();
            } catch {
                // already closing
            }
            this.ws = null;
        }
        this.buffer = null;
        this.bufferCtx = null;
        this.display = null;
        this.displayCtx = null;
        this.resizeObserver?.disconnect();
        this.resizeObserver = null;
    }

    private scheduleReconnect() {
        if (this.disposed || this.paused) return;
        this.clearReconnect();
        this.reconnectTimer = window.setTimeout(() => {
            this.reconnectTimer = null;
            this.connect();
        }, RECONNECT_MS);
    }

    private clearReconnect() {
        if (this.reconnectTimer !== null) {
            window.clearTimeout(this.reconnectTimer);
            this.reconnectTimer = null;
        }
    }

    private setConnection(state: ConnectionState) {
        this.connection = state;
        this.handlers.onConnection?.(state);
    }

    // syncControl derives the control state from the holder reported by the
    // server. holderOverride covers a control message whose status snapshot has
    // not been applied yet.
    private syncControl(holderOverride?: string) {
        if (holderOverride === undefined && !this.status?.control) return;
        const holder = holderOverride ?? String(this.status?.control?.holder ?? '');
        if (holder && holder === this.viewerId) {
            this.setControl('active');
        } else if (holder) {
            this.setControl('locked');
        } else {
            this.setControl('view');
        }
    }

    private setControl(state: ControlState) {
        if (this.control === state) return;
        this.control = state;
        this.handlers.onControl?.(state);
    }

    private send(data: ArrayBuffer) {
        if (this.ws && this.ws.readyState === WebSocket.OPEN) {
            this.ws.send(data);
        }
    }

    private sendJSON(message: unknown) {
        if (this.ws && this.ws.readyState === WebSocket.OPEN) {
            this.ws.send(JSON.stringify(message));
        }
    }

    // ---------------------------------------------------------------- canvas

    attach(canvas: HTMLCanvasElement) {
        this.display = canvas;
        this.displayCtx = canvas.getContext('2d');
        // The stage changes size on rotation, when the key bar is toggled and
        // when immersive mode hides the chrome, so re-fit instead of waiting for
        // the next frame from the Mac.
        this.resizeObserver?.disconnect();
        if (typeof ResizeObserver !== 'undefined') {
            this.resizeObserver = new ResizeObserver(() => this.scheduleBlit());
            this.resizeObserver.observe(canvas);
        }
        this.blit();
    }

    detach(canvas: HTMLCanvasElement) {
        if (this.display === canvas) {
            this.resizeObserver?.disconnect();
            this.resizeObserver = null;
            this.display = null;
            this.displayCtx = null;
        }
    }

    get imageSize() {
        return { width: this.imgW, height: this.imgH };
    }

    private ensureBuffer(w: number, h: number) {
        if (this.buffer && this.buffer.width === w && this.buffer.height === h) return;
        const previous = { w: this.imgW, h: this.imgH };
        const canvas = document.createElement('canvas');
        canvas.width = w;
        canvas.height = h;
        this.buffer = canvas;
        this.bufferCtx = canvas.getContext('2d');
        this.imgW = w;
        this.imgH = h;
        // Keep looking at the same part of the Mac when the stream is re-scaled
        // (a display change, or a reconnect at a different resolution).
        if (!this.viewReady) {
            this.view = centreView(this.geometry());
            this.viewReady = true;
        } else if (previous.w && previous.h) {
            this.view = {
                zoom: this.view.zoom,
                cx: (this.view.cx / previous.w) * w,
                cy: (this.view.cy / previous.h) * h,
            };
        }
        this.emitView();
    }

    // ---------------------------------------------------------------- viewport

    private view: Viewport = { zoom: 1, cx: 0, cy: 0 };
    private viewReady = false;
    private showCursor = true;
    private resizeObserver: ResizeObserver | null = null;

    /** Stage and image sizes in CSS pixels; null until the canvas is attached. */
    private geometry(): ViewGeometry {
        const canvas = this.display;
        return {
            stageW: canvas?.clientWidth || 0,
            stageH: canvas?.clientHeight || 0,
            imgW: this.imgW,
            imgH: this.imgH,
        };
    }

    private emitView() {
        this.handlers.onView?.({ ...this.view });
    }

    get viewport(): Viewport {
        return { ...this.view };
    }

    get zoom(): number {
        return this.view.zoom;
    }

    setShowCursor(visible: boolean) {
        if (this.showCursor === visible) return;
        this.showCursor = visible;
        this.scheduleBlit();
    }

    /** Applies a view, clamped to the image, and repaints from the buffer. */
    setView(next: Viewport) {
        this.view = clampView(this.geometry(), next);
        this.viewReady = true;
        this.emitView();
        this.scheduleBlit();
    }

    /** Zooms so the given image pixel stays under the same screen pixel. */
    zoomAt(zoom: number, anchor?: Point) {
        const g = this.geometry();
        const centre = { x: g.imgW / 2, y: g.imgH / 2 };
        this.setView(zoomAt(g, this.view, zoom, anchor ?? centre));
    }

    zoomBy(direction: 1 | -1) {
        this.zoomAt(stepZoom(this.view.zoom, direction));
    }

    zoomToggle() {
        this.zoomAt(toggleZoom(this.view.zoom));
    }

    fit() {
        this.setView(centreView(this.geometry()));
    }

    /** Pan by a screen-space delta (CSS pixels), as a drag does. */
    panByScreen(dx: number, dy: number) {
        const g = this.geometry();
        this.setView(panBy(g, this.view, dx, dy));
    }

    /** Converts a viewport point (clientX/clientY) to stage CSS pixels. */
    private stagePoint(clientX: number, clientY: number): Point | null {
        const canvas = this.display;
        if (!canvas) return null;
        const rect = canvas.getBoundingClientRect();
        return { x: clientX - rect.left, y: clientY - rect.top };
    }

    /** Image pixel under a viewport point, unclamped — used as a pinch anchor. */
    streamAt(clientX: number, clientY: number): Point | null {
        const g = this.geometry();
        const point = this.stagePoint(clientX, clientY);
        if (!point || !g.imgW || !g.imgH) return null;
        return screenToStream(g, this.view, point);
    }

    /**
     * Zooms to `zoom` and centres the view so the image pixel `anchor` sits
     * exactly under the given viewport point. This is the pinch primitive: it
     * handles magnification and translation in one step, so a pinch never
     * drifts away from the fingers.
     */
    placeStreamAt(anchorX: number, anchorY: number, clientX: number, clientY: number, zoom = this.view.zoom) {
        const g = this.geometry();
        const point = this.stagePoint(clientX, clientY);
        if (!point || !g.imgW || !g.imgH) return;
        const k = scaleOf(g, { zoom, cx: 0, cy: 0 });
        if (!k) return;
        this.setView({
            zoom,
            cx: anchorX - (point.x - g.stageW / 2) / k,
            cy: anchorY - (point.y - g.stageH / 2) / k,
        });
    }

    /** Zooms around a raw event position, used by pinch and double tap. */
    zoomAtClient(zoom: number, clientX: number, clientY: number) {
        const g = this.geometry();
        const point = this.stagePoint(clientX, clientY);
        if (!point || !g.imgW) {
            this.zoomAt(zoom);
            return;
        }
        const anchor = {
            x: this.view.cx + (point.x - g.stageW / 2) / scaleOf(g, this.view),
            y: this.view.cy + (point.y - g.stageH / 2) / scaleOf(g, this.view),
        };
        this.setView(zoomAt(g, this.view, zoom, anchor));
    }

    zoomByClient(direction: 1 | -1, clientX: number, clientY: number) {
        this.zoomAtClient(stepZoom(this.view.zoom, direction), clientX, clientY);
    }

    zoomToggleClient(clientX: number, clientY: number) {
        this.zoomAtClient(toggleZoom(this.view.zoom), clientX, clientY);
    }

    private scheduleBlit() {
        if (this.rafPending) return;
        this.rafPending = true;
        requestAnimationFrame(() => {
            this.rafPending = false;
            this.blit();
        });
    }

    private blit() {
        const canvas = this.display;
        const ctx = this.displayCtx;
        const buffer = this.buffer;
        if (!canvas || !ctx) return;

        const dpr = window.devicePixelRatio || 1;
        const cssW = canvas.clientWidth || canvas.width;
        const cssH = canvas.clientHeight || canvas.height;
        const backingW = Math.max(1, Math.round(cssW * dpr));
        const backingH = Math.max(1, Math.round(cssH * dpr));
        if (canvas.width !== backingW || canvas.height !== backingH) {
            canvas.width = backingW;
            canvas.height = backingH;
        }

        // Draw in CSS pixels: the viewport maths and the pointer mapping then
        // agree on one coordinate system, and the device pixel ratio only ever
        // lives in this transform.
        ctx.setTransform(dpr, 0, 0, dpr, 0, 0);
        ctx.fillStyle = '#0b0d10';
        ctx.fillRect(0, 0, cssW, cssH);
        if (!buffer || !this.imgW || !this.imgH) return;

        const g = this.geometry();
        if (!g.stageW || !g.stageH) return;
        const settled = clampView(g, this.view);
        if (settled.zoom !== this.view.zoom || settled.cx !== this.view.cx || settled.cy !== this.view.cy) {
            // The stage changed size under us (rotation, key bar, immersive).
            this.view = settled;
            this.emitView();
        }
        const rect = drawRectOf(g, settled);
        if (!(rect.sw > 0) || !(rect.sh > 0)) return;
        ctx.imageSmoothingEnabled = true;
        ctx.imageSmoothingQuality = 'high';
        if (rect.sx === 0 && rect.sy === 0 && rect.sw === this.imgW && rect.sh === this.imgH) {
            ctx.drawImage(buffer, rect.dx, rect.dy, rect.dw, rect.dh);
        } else {
            ctx.drawImage(buffer, rect.sx, rect.sy, rect.sw, rect.sh, rect.dx, rect.dy, rect.dw, rect.dh);
        }

        if (this.cursor.visible && this.showCursor) {
            const k = scaleOf(g, settled);
            const point = streamToScreen(g, settled, { x: this.cursor.x, y: this.cursor.y });
            this.drawCursor(ctx, point.x, point.y, k);
        }
    }

    /** Maps a pointer event position to image pixel coordinates. */
    toImageCoords(event: { clientX: number; clientY: number }): { x: number; y: number } | null {
        const canvas = this.display;
        if (!canvas || !this.imgW || !this.imgH) return null;
        const point = this.stagePoint(event.clientX, event.clientY);
        if (!point) return null;
        return screenToImage(this.geometry(), this.view, point);
    }

    /** Size of the arrow in CSS pixels: readable on a phone, not huge when zoomed. */
    private drawCursor(ctx: CanvasRenderingContext2D, x: number, y: number, k: number) {
        const size = Math.max(12, Math.min(28, Math.round(18 * k)));
        ctx.save();
        ctx.translate(x, y);
        ctx.scale(size / 18, size / 18);
        ctx.beginPath();
        ctx.moveTo(0, 0);
        ctx.lineTo(0, 15);
        ctx.lineTo(4, 11.5);
        ctx.lineTo(6.8, 17.5);
        ctx.lineTo(9.6, 16);
        ctx.lineTo(6.9, 10.2);
        ctx.lineTo(12, 10);
        ctx.closePath();
        ctx.fillStyle = this.cursor.buttons ? '#ffd166' : '#ffffff';
        ctx.strokeStyle = '#000000';
        ctx.lineWidth = 1.2;
        ctx.fill();
        ctx.stroke();
        ctx.restore();
    }

    // -------------------------------------------------------------- messages

    private handleText(raw: string) {
        let message: Record<string, unknown>;
        try {
            message = JSON.parse(raw);
        } catch {
            return;
        }
        switch (message.type) {
            case 'hello':
                this.viewerId = String(message.viewer_id ?? '');
                if (message.status) {
                    this.status = message.status as ScreenStatus;
                    this.handlers.onStatus?.(this.status);
                    this.syncControl();
                }
                break;
            case 'status':
                if (message.status) {
                    this.status = message.status as ScreenStatus;
                    this.handlers.onStatus?.(this.status);
                    this.syncControl();
                }
                break;
            case 'control': {
                if (message.granted) {
                    this.releasing = false;
                    this.setControl('active');
                    break;
                }
                // The holder in the denying message is authoritative: a viewer
                // that just released control must fall back to "view", not
                // "locked" (nothing is held anymore).
                this.syncControl(String(message.holder ?? ''));
                const released = String(message.viewer_id ?? '');
                const mine = released !== '' && released === this.viewerId;
                // A release broadcast names the viewer it affected, a denial
                // names none. Stay quiet about a release this viewer asked for,
                // but surface an involuntary one (idle timeout, locked host).
                if (message.reason && (released === '' || (mine && !this.releasing))) {
                    this.handlers.onError?.(String(message.reason), 'released');
                }
                if (mine) this.releasing = false;
                break;
            }
            case 'error':
                this.handlers.onError?.(
                    String(message.message ?? 'screen error'),
                    typeof message.code === 'string' ? message.code : undefined,
                );
                break;
            case 'perm':
                if (message.input && this.status) {
                    this.status = { ...this.status, input: message.input as ScreenStatus['input'] };
                    this.handlers.onStatus?.(this.status);
                }
                break;
            default:
                break;
        }
    }

    private handleBinary(data: ArrayBuffer) {
        if (data.byteLength < 1) return;
        const view = new DataView(data);
        const type = view.getUint8(0);
        if (type === MSG_CURSOR) {
            if (data.byteLength < 8) return;
            this.cursor = {
                visible: view.getUint8(1) === 1,
                shape: view.getUint8(2),
                buttons: view.getUint8(3),
                x: view.getUint16(4, true),
                y: view.getUint16(6, true),
            };
            this.scheduleBlit();
            return;
        }
        if (type === MSG_FRAME) {
            this.queue = this.queue.then(() => this.applyFrame(data)).catch(() => undefined);
        }
    }

    private async applyFrame(data: ArrayBuffer) {
        if (data.byteLength < FRAME_HEADER) return;
        const view = new DataView(data);
        const imgW = view.getUint16(12, true);
        const imgH = view.getUint16(14, true);
        const count = view.getUint16(20, true);
        if (!imgW || !imgH) return;
        this.ensureBuffer(imgW, imgH);
        const ctx = this.bufferCtx;
        if (!ctx) return;

        let offset = FRAME_HEADER;
        const decoded: Array<{ bitmap: ImageBitmap; x: number; y: number }> = [];
        for (let i = 0; i < count; i++) {
            if (offset + TILE_HEADER > data.byteLength) break;
            const x = view.getUint16(offset, true);
            const y = view.getUint16(offset + 2, true);
            const len = view.getUint32(offset + 4, true);
            offset += TILE_HEADER;
            if (offset + len > data.byteLength) break;
            const bytes = new Uint8Array(data, offset, len);
            offset += len;
            const blob = new Blob([bytes], { type: 'image/jpeg' });
            try {
                const bitmap = await createImageBitmap(blob);
                decoded.push({ bitmap, x, y });
            } catch {
                // a truncated tile just leaves the previous pixels in place
            }
        }
        for (const tile of decoded) {
            ctx.drawImage(tile.bitmap, tile.x, tile.y);
            tile.bitmap.close();
        }
        this.scheduleBlit();
    }

    // ----------------------------------------------------------------- input

    requestControl() {
        this.sendJSON({ type: 'request_control' });
    }

    releaseControl() {
        this.releaseAll();
        this.releasing = true;
        this.sendJSON({ type: 'release_control' });
        this.setControl('view');
    }

    releaseAll() {
        this.send(new Uint8Array([MSG_RELEASE_ALL]).buffer);
    }

    /** Last image-space position we sent; used for click buttons on touch UIs. */
    lastPoint = { x: 0, y: 0 };

    clickAt(button: number) {
        const { x, y } = this.lastPoint;
        this.buttonDown(button, x, y);
        this.buttonUp(button, x, y);
    }

    private pointerBuffer(action: number, button: number, mods: number, x: number, y: number, aux = 0): ArrayBuffer {
        const buf = new ArrayBuffer(10);
        const view = new DataView(buf);
        view.setUint8(0, MSG_POINTER);
        view.setUint8(1, action);
        view.setUint8(2, button);
        view.setUint8(3, mods);
        view.setUint16(4, Math.max(0, Math.min(65535, x)), true);
        view.setUint16(6, Math.max(0, Math.min(65535, y)), true);
        view.setInt16(8, aux, true);
        return buf;
    }

    move(x: number, y: number, mods = 0) {
        this.lastPoint = { x, y };
        this.send(this.pointerBuffer(POINTER_MOVE, 0, mods, x, y));
    }

    buttonDown(button: number, x: number, y: number, mods = 0) {
        this.lastPoint = { x, y };
        this.send(this.pointerBuffer(POINTER_DOWN, button, mods, x, y));
    }

    buttonUp(button: number, x: number, y: number, mods = 0) {
        this.lastPoint = { x, y };
        this.send(this.pointerBuffer(POINTER_UP, button, mods, x, y));
    }

    /** dy is in wheel lines (positive = towards the user), dx folds into it. */
    wheel(dx: number, dy: number, x: number, y: number) {
        let lines = Math.trunc(dy);
        if (!lines && dx) lines = Math.sign(dx);
        if (!lines) return;
        this.send(this.pointerBuffer(POINTER_WHEEL, 0, 0, x, y, lines));
    }

    key(code: string, down: boolean) {
        const bytes = new TextEncoder().encode(code);
        const buf = new ArrayBuffer(4 + bytes.length);
        const view = new DataView(buf);
        view.setUint8(0, MSG_KEY);
        view.setUint8(1, down ? 1 : 0);
        view.setUint16(2, bytes.length, true);
        new Uint8Array(buf, 4).set(bytes);
        this.send(buf);
    }

    text(value: string) {
        const bytes = new TextEncoder().encode(value);
        const buf = new ArrayBuffer(3 + bytes.length);
        const view = new DataView(buf);
        view.setUint8(0, MSG_TEXT);
        view.setUint16(1, bytes.length, true);
        new Uint8Array(buf, 3).set(bytes);
        this.send(buf);
    }
}
