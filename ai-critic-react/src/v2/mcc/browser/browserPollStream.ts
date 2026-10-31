import type { BrowserMode, BrowserStatus, ClientMessage } from './browserWire';

/**
 * Frame intervals the poll menu offers, in milliseconds. Balanced is the
 * default: one frame per 3 s reads as "live enough" on 4G without draining
 * the battery.
 */
export const POLL_INTERVALS = [1000, 3000, 5000] as const;
export type PollInterval = (typeof POLL_INTERVALS)[number];
export const DEFAULT_POLL_INTERVAL: PollInterval = 3000;

const SLOW_RTT_MS = 3000;
const STATUS_EVERY_MS = 5000;

/**
 * One HTTP poll loop: the transport for a network where the WebSocket cannot
 * connect. GET /api/browser/frame?after=seq answers 304 (nothing changed —
 * zero bytes) or 200 image/jpeg with the newest seq in X-Frame-Seq; status
 * comes from /api/browser/status, input and commands from the REST endpoints.
 *
 * Same surface as BrowserStream, so the page swaps transports without caring.
 */
export class BrowserPollStream {
    private readonly handlers: import('./browserStream').BrowserStreamHandlers;
    private img: HTMLImageElement | null = null;
    private frameUrl: string | null = null;
    private disposed = false;
    private paused = false;
    private timer: ReturnType<typeof setTimeout> | null = null;
    private statusTimer: ReturnType<typeof setInterval> | null = null;
    private lastSeq = 0;
    private interval: PollInterval;
    private lastFrameAt = 0;
    private inflight = false;
    private pendingStatus: BrowserStatus | null = null;

    constructor(handlers: import('./browserStream').BrowserStreamHandlers = {}, interval: PollInterval = DEFAULT_POLL_INTERVAL) {
        this.handlers = handlers;
        this.interval = interval;
    }

    get frameAge(): number {
        return this.lastFrameAt === 0 ? Number.POSITIVE_INFINITY : Date.now() - this.lastFrameAt;
    }

    /** Test hook: the current interval in ms. */
    get intervalMs(): number {
        return this.interval;
    }

    attachImage(img: HTMLImageElement | null) {
        this.img = img;
    }

    connect() {
        if (this.disposed || this.paused) return;
        this.handlers.onConnection?.('connecting');
        this.schedule(0);
        this.statusTimer = setInterval(() => void this.fetchStatus(), STATUS_EVERY_MS);
        void this.fetchStatus();
    }

    pause() {
        this.paused = true;
        this.clearTimers();
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

    /** Input and control ride REST; see browserApi.ts for the endpoints. */
    send(msg: ClientMessage) {
        void this.dispatch(msg);
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
        void this.fetchStatus();
    }

    get status(): BrowserStatus | null {
        return this.pendingStatus;
    }

    /** Test hook: the newest seq the poll has seen. */
    get seq(): number {
        return this.lastSeq;
    }

    // -------------------------------------------------------------- internals

    private schedule(delay: number) {
        if (this.disposed || this.paused) return;
        if (this.timer) clearTimeout(this.timer);
        this.timer = setTimeout(() => void this.tick(), delay);
    }

    private async tick() {
        if (this.disposed || this.paused || this.inflight) return;
        this.inflight = true;
        const started = Date.now();
        try {
            const response = await fetch(`/api/browser/frame?after=${this.lastSeq}`, { cache: 'no-store' });
            const rtt = Date.now() - started;
            if (response.status === 200) {
                const seqHeader = response.headers.get('X-Frame-Seq');
                const blob = await response.blob();
                if (seqHeader) this.lastSeq = Number(seqHeader);
                if (blob.size > 0) this.paint(blob);
            }
            // A round trip slower than the interval means the network just got
            // worse: back off instead of piling requests onto a struggling link.
            this.interval = nextInterval(this.interval, rtt);
            this.handlers.onConnection?.('open');
        } catch {
            this.handlers.onConnection?.('closed');
            this.handlers.onError?.('poll request failed');
        } finally {
            this.inflight = false;
            if (!this.disposed && !this.paused) this.schedule(this.interval);
        }
    }

    private async fetchStatus() {
        if (this.disposed || this.paused) return;
        try {
            const response = await fetch('/api/browser/status', { cache: 'no-store' });
            if (!response.ok) return;
            const status = (await response.json()) as BrowserStatus;
            this.pendingStatus = status;
            this.handlers.onStatus?.(status);
        } catch {
            /* the next tick retries */
        }
    }

    private async dispatch(msg: ClientMessage) {
        const endpoint = controlEndpoint(msg);
        try {
            if (endpoint) {
                const response = await fetch(endpoint.path, {
                    method: 'POST',
                    headers: { 'Content-Type': 'application/json' },
                    body: JSON.stringify(endpoint.body),
                });
                if (!response.ok) {
                    const detail = await response.json().catch(() => null);
                    const message = detail && typeof detail === 'object' && 'error' in detail ? String((detail as { error?: unknown }).error) : `request failed (${response.status})`;
                    this.handlers.onError?.(message);
                    return;
                }
                const status = (await response.json()) as BrowserStatus;
                this.pendingStatus = status;
                this.handlers.onStatus?.(status);
            } else {
                // Pointer, wheel, key, text: one event per POST.
                const response = await fetch('/api/browser/input', {
                    method: 'POST',
                    headers: { 'Content-Type': 'application/json' },
                    body: JSON.stringify(msg),
                });
                if (!response.ok) {
                    this.handlers.onError?.(`input failed (${response.status})`);
                }
            }
        } catch {
            this.handlers.onError?.('poll request failed');
        }
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

    private clearTimers() {
        if (this.timer) {
            clearTimeout(this.timer);
            this.timer = null;
        }
        if (this.statusTimer) {
            clearInterval(this.statusTimer);
            this.statusTimer = null;
        }
    }
}

/** Back off when the link is slower than the interval; never speeds back up on
 * its own — the menu owns that choice, and a healed network at 5 s is just a
 * calm poll. */
export function nextInterval(current: PollInterval, rttMs: number): PollInterval {
    const idx = POLL_INTERVALS.indexOf(current);
    if (rttMs > SLOW_RTT_MS && idx >= 0 && idx < POLL_INTERVALS.length - 1) {
        return POLL_INTERVALS[idx + 1];
    }
    return current;
}

/** REST equivalent for every control message the WS transport accepted. */
function controlEndpoint(msg: ClientMessage): { path: string; body: Record<string, unknown> } | null {
    switch (msg.type) {
        case 'navigate':
            return { path: '/api/browser/navigate', body: { url: msg.url } };
        case 'preset':
            return { path: '/api/browser/preset', body: { preset: msg.preset } };
        case 'mode':
            return { path: '/api/browser/mode', body: { mode: msg.mode } };
        case 'refresh':
            return { path: '/api/browser/refresh', body: {} };
        default:
            return null;
    }
}
