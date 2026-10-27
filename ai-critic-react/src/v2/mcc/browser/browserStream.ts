import { FRAME_HEADER_SIZE, FRAME_MAGIC } from './browserWire';
import type { BrowserMode, BrowserStatus, ClientMessage, ConnectionState } from './browserWire';

export interface BrowserStreamHandlers {
    onStatus?: (status: BrowserStatus) => void;
    onConnection?: (state: ConnectionState) => void;
    onError?: (message: string) => void;
    /** Fired after every painted frame, for stall detection. */
    onFrame?: () => void;
}

/**
 * One /api/browser/ws connection: decodes frames into an <img>, keeps the socket
 * alive, and reconnects when it goes quiet.
 *
 * Frames are JPEGs of a screenshot, not a live screencast, so the client paints
 * the newest one and drops the rest. Coordinates are normalized, so the page
 * never needs to know which viewport the window actually achieved.
 */
export class BrowserStream {
    private readonly handlers: BrowserStreamHandlers;
    private ws: WebSocket | null = null;
    private img: HTMLImageElement | null = null;
    private frameUrl: string | null = null;
    private disposed = false;
    private paused = false;
    private reconnectTimer: ReturnType<typeof setTimeout> | null = null;
    private watchdog: ReturnType<typeof setInterval> | null = null;
    private lastMessageAt = 0;
    private reconnectDelay = 1000;
    private lastFrameAt = 0;
    private pendingStatus: BrowserStatus | null = null;

    constructor(handlers: BrowserStreamHandlers = {}) {
        this.handlers = handlers;
    }

    /** Milliseconds since the last painted frame. */
    get frameAge(): number {
        return this.lastFrameAt === 0 ? Number.POSITIVE_INFINITY : Date.now() - this.lastFrameAt;
    }

    attachImage(img: HTMLImageElement | null) {
        this.img = img;
    }

    connect() {
        if (this.disposed || this.paused) return;
        this.clearReconnect();
        const protocol = window.location.protocol === 'https:' ? 'wss:' : 'ws:';
        const url = `${protocol}//${window.location.host}/api/browser/ws`;
        this.handlers.onConnection?.('connecting');
        const ws = new WebSocket(url);
        ws.binaryType = 'arraybuffer';
        this.ws = ws;
        this.lastMessageAt = Date.now();
        this.startWatchdog();

        ws.onopen = () => {
            if (this.ws !== ws) return;
            this.reconnectDelay = 1000;
            this.handlers.onConnection?.('open');
        };
        ws.onclose = () => {
            if (this.ws !== ws) return;
            this.ws = null;
            this.handlers.onConnection?.('closed');
            this.scheduleReconnect();
        };
        ws.onerror = () => {
            this.handlers.onError?.('browser connection failed');
        };
        ws.onmessage = (event) => {
            if (this.ws !== ws) return;
            this.lastMessageAt = Date.now();
            if (typeof event.data === 'string') this.handleText(event.data);
            else this.handleBinary(event.data as ArrayBuffer);
        };
    }

    /** Closing the last socket is what stops capture on the host. */
    pause() {
        this.paused = true;
        this.clearReconnect();
        this.stopWatchdog();
        const ws = this.ws;
        this.ws = null;
        if (ws) {
            try {
                ws.close();
            } catch {
                /* already closing */
            }
        }
    }

    resume() {
        if (this.disposed || !this.paused) return;
        this.paused = false;
        this.connect();
    }

    dispose() {
        this.disposed = true;
        this.pause();
        if (this.frameUrl) {
            URL.revokeObjectURL(this.frameUrl);
            this.frameUrl = null;
        }
        this.img = null;
    }

    send(msg: ClientMessage) {
        const ws = this.ws;
        if (!ws || ws.readyState !== WebSocket.OPEN) return;
        ws.send(JSON.stringify(msg));
    }

    navigate(url: string) {
        this.send({ type: 'navigate', url });
    }

    setPreset(preset: string) {
        this.send({ type: 'preset', preset });
    }

    setMode(mode: BrowserMode) {
        this.send({ type: 'mode', mode });
    }

    refresh() {
        this.send({ type: 'refresh' });
    }

    requestStatus() {
        this.send({ type: 'status' });
    }

    /** Status from the last message, for a page that mounts after connect. */
    get status(): BrowserStatus | null {
        return this.pendingStatus;
    }

    // -------------------------------------------------------------- internals

    private handleText(raw: string) {
        let payload: { type?: string; status?: BrowserStatus; message?: string };
        try {
            payload = JSON.parse(raw);
        } catch {
            return;
        }
        switch (payload.type) {
            case 'status':
                if (payload.status) {
                    this.pendingStatus = payload.status;
                    this.handlers.onStatus?.(payload.status);
                }
                break;
            case 'hello':
                if (payload.status) {
                    this.pendingStatus = payload.status;
                    this.handlers.onStatus?.(payload.status);
                }
                break;
            case 'error':
                if (payload.message) this.handlers.onError?.(payload.message);
                break;
            default:
                break;
        }
    }

    private handleBinary(buffer: ArrayBuffer) {
        if (buffer.byteLength <= FRAME_HEADER_SIZE) return;
        const view = new DataView(buffer);
        if (view.getUint8(0) !== FRAME_MAGIC) return;
        const length = view.getUint32(12);
        if (length === 0 || FRAME_HEADER_SIZE + length > buffer.byteLength) return;
        const jpeg = new Blob([buffer.slice(FRAME_HEADER_SIZE, FRAME_HEADER_SIZE + length)], {
            type: 'image/jpeg',
        });
        this.paint(jpeg);
    }

    private paint(jpeg: Blob) {
        const img = this.img;
        if (!img) return;
        const url = URL.createObjectURL(jpeg);
        const previous = this.frameUrl;
        this.frameUrl = url;
        img.onload = () => {
            if (previous) URL.revokeObjectURL(previous);
        };
        img.src = url;
        this.lastFrameAt = Date.now();
        this.handlers.onFrame?.();
    }

    private startWatchdog() {
        this.stopWatchdog();
        // Keepalives arrive at least every 3 s, so 20 s of silence means the
        // socket is dead even though the browser has not noticed.
        this.watchdog = setInterval(() => {
            if (Date.now() - this.lastMessageAt > 20000) {
                this.dropAndReconnect();
            }
        }, 5000);
    }

    private stopWatchdog() {
        if (this.watchdog) {
            clearInterval(this.watchdog);
            this.watchdog = null;
        }
    }

    private dropAndReconnect() {
        const ws = this.ws;
        this.ws = null;
        this.stopWatchdog();
        if (ws) {
            try {
                ws.close();
            } catch {
                /* already gone */
            }
        }
        this.handlers.onConnection?.('closed');
        this.scheduleReconnect();
    }

    private scheduleReconnect() {
        if (this.disposed || this.paused || this.reconnectTimer) return;
        const delay = this.reconnectDelay;
        this.reconnectDelay = Math.min(this.reconnectDelay * 2, 8000);
        this.reconnectTimer = setTimeout(() => {
            this.reconnectTimer = null;
            this.connect();
        }, delay);
    }

    private clearReconnect() {
        if (this.reconnectTimer) {
            clearTimeout(this.reconnectTimer);
            this.reconnectTimer = null;
        }
    }
}
