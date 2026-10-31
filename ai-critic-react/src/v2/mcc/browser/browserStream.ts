import { FRAME_HEADER_SIZE, FRAME_MAGIC } from './browserWire';
import type { BrowserMode, BrowserStatus, ClientMessage, ConnectionState } from './browserWire';

/** How long a connection attempt may stay unanswered before probing REST. */
export const CONNECT_PROBE_MS = 6000;
/** Consecutive reconnects without reaching open that count as "blocked". */
export const PROBE_AFTER_FAILURES = 3;
/** Silence threshold for "frames stopped but the socket says open". */
export const STALL_PROBE_MS = 30_000;

export interface ProbeEvidence {
    /** WebSocket connect attempts since the last success. */
    attempts: number;
    /** REST /api/browser/status round trip, when the probe measured one. */
    restMs: number | null;
    /** Milliseconds since the last painted frame. */
    frameAge: number;
    /** Which trigger fired the probe. */
    reason: 'connect-timeout' | 'reconnect-failures' | 'stalled';
}

export interface BrowserStreamHandlers {
    onStatus?: (status: BrowserStatus) => void;
    onConnection?: (state: ConnectionState) => void;
    onError?: (message: string) => void;
    /** Fired after every painted frame, for stall detection. */
    onFrame?: () => void;
    /**
     * The transport looks broken while the host may be fine: the page offers
     * the poll fallback with this evidence. Fires once per episode; the page
     * calls resetProbe() after switching transports or dismissing the banner.
     */
    onProbe?: (evidence: ProbeEvidence) => void;
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
    // Probe bookkeeping: attempts counts connects since the last open,
    // probeFired keeps the banner from repeating inside one bad episode.
    private attempts = 0;
    private probeFired = false;
    private connectTimer: ReturnType<typeof setTimeout> | null = null;

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
        this.attempts++;
        const ws = new WebSocket(url);
        ws.binaryType = 'arraybuffer';
        this.ws = ws;
        this.lastMessageAt = Date.now();
        this.startWatchdog();
        this.armConnectProbe();

        ws.onopen = () => {
            if (this.ws !== ws) return;
            this.reconnectDelay = 1000;
            this.attempts = 0;
            this.disarmConnectProbe();
            this.handlers.onConnection?.('open');
        };
        ws.onclose = () => {
            if (this.ws !== ws) return;
            this.ws = null;
            this.disarmConnectProbe();
            this.handlers.onConnection?.('closed');
            if (this.attempts >= PROBE_AFTER_FAILURES) {
                this.fireProbe('reconnect-failures');
            }
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
        this.disarmConnectProbe();
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
                return;
            }
            // Open but silent on frames: a network that slow deserves the same
            // offer as a blocked socket.
            if (this.lastFrameAt > 0 && Date.now() - this.lastFrameAt > STALL_PROBE_MS) {
                this.fireProbe('stalled');
            }
        }, 5000);
    }

    // ------------------------------------------------------------- probing

    /** One missed handshake: measure REST and let the page offer poll mode. */
    private armConnectProbe() {
        this.disarmConnectProbe();
        this.connectTimer = setTimeout(() => {
            this.connectTimer = null;
            if (this.ws) this.fireProbe('connect-timeout');
        }, CONNECT_PROBE_MS);
    }

    private disarmConnectProbe() {
        if (this.connectTimer) {
            clearTimeout(this.connectTimer);
            this.connectTimer = null;
        }
    }

    private fireProbe(reason: ProbeEvidence['reason']) {
        if (this.probeFired || this.disposed || this.paused) return;
        this.probeFired = true;
        const started = Date.now();
        const controller = new AbortController();
        const bail = setTimeout(() => controller.abort(), 3000);
        fetch('/api/browser/status', { signal: controller.signal })
            .then((response) => {
                clearTimeout(bail);
                this.handlers.onProbe?.({
                    attempts: this.attempts,
                    restMs: response.ok ? Date.now() - started : null,
                    frameAge: this.frameAge === Number.POSITIVE_INFINITY ? -1 : this.frameAge,
                    reason,
                });
            })
            .catch(() => {
                clearTimeout(bail);
                this.handlers.onProbe?.({
                    attempts: this.attempts,
                    restMs: null,
                    frameAge: this.frameAge === Number.POSITIVE_INFINITY ? -1 : this.frameAge,
                    reason,
                });
            });
    }

    /** A new episode: the user retried WS or dismissed the banner. */
    resetProbe() {
        this.probeFired = false;
        this.attempts = 0;
    }

    /** Test hook: whether the probe already fired in this episode. */
    get probed(): boolean {
        return this.probeFired;
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
