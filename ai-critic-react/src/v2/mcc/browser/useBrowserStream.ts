import { useEffect, useRef, useState } from 'react';
import { BrowserStream } from './browserStream';
import type { BrowserStreamHandlers, ProbeEvidence } from './browserStream';
import { BrowserPollStream, DEFAULT_POLL_INTERVAL, POLL_INTERVALS } from './browserPollStream';
import type { PollInterval } from './browserPollStream';
import { BrowserGestures } from './browserGestures';
import type { GestureHost } from './browserGestures';
import type { BrowserMode, BrowserStatus, ClientMessage, ConnectionState } from './browserWire';

/** Which transport paints the picture. Persisted per device. */
export type StreamTransport = 'ws' | 'poll';

const TRANSPORT_KEY = 'ai-critic-stream-mode';
const INTERVAL_KEY = 'ai-critic-poll-interval';

function loadTransport(): StreamTransport {
    try {
        return localStorage.getItem(TRANSPORT_KEY) === 'poll' ? 'poll' : 'ws';
    } catch {
        return 'ws';
    }
}

function loadInterval(): PollInterval {
    try {
        const raw = Number(localStorage.getItem(INTERVAL_KEY));
        return (POLL_INTERVALS as readonly number[]).includes(raw) ? (raw as PollInterval) : DEFAULT_POLL_INTERVAL;
    } catch {
        return DEFAULT_POLL_INTERVAL;
    }
}

function persist(transport: StreamTransport, interval: PollInterval) {
    try {
        localStorage.setItem(TRANSPORT_KEY, transport);
        localStorage.setItem(INTERVAL_KEY, String(interval));
    } catch {
        /* private mode: session-only choice */
    }
}

/**
 * Owns the transport and the gesture translator for the lifetime of the page.
 *
 * The default transport is one WebSocket. When the probe decides the socket is
 * blocked or starved while the host answers REST, the page offers poll mode;
 * switching swaps the whole stream object, so the rest of the page neither
 * knows nor cares which transport is live.
 *
 * Unlike /screen there is no control handshake: every viewer may click. The
 * browser window is a shared surface, and a second viewer taking over mid-click
 * is less surprising than a modal nobody can dismiss on a phone.
 */
export function useBrowserStream() {
    const streamRef = useRef<BrowserStream | null>(null);
    const pollRef = useRef<BrowserPollStream | null>(null);
    const gesturesRef = useRef<BrowserGestures | null>(null);
    const inputEnabled = useRef(false);
    const stageRef = useRef<HTMLDivElement | null>(null);
    const imgRef = useRef<HTMLImageElement | null>(null);
    const cursorRef = useRef<HTMLDivElement | null>(null);
    const [status, setStatus] = useState<BrowserStatus | null>(null);
    const [connection, setConnection] = useState<ConnectionState>('connecting');
    const [error, setError] = useState<string | null>(null);
    const [dragging, setDragging] = useState(false);
    const [fallbackNote, setFallbackNote] = useState<string | null>(null);
    const fallbackDone = useRef(false);
    const [transport, setTransport] = useState<StreamTransport>(loadTransport);
    const [pollInterval, setPollIntervalState] = useState<PollInterval>(loadInterval);
    const [probe, setProbe] = useState<ProbeEvidence | null>(null);
    // The live stream behind gestures and commands; both shapes expose the
    // same control surface.
    const activeRef = useRef<
        Pick<BrowserStream, 'send' | 'navigate' | 'setPreset' | 'setMode' | 'refresh' | 'frameAge'> | null
    >(null);

    const makeHandlers = (withProbe: boolean): BrowserStreamHandlers => ({
        onStatus: setStatus,
        onConnection: setConnection,
        onError: (message) => setError(message),
        ...(withProbe ? { onProbe: (evidence: ProbeEvidence) => setProbe(evidence) } : {}),
    });

    const startTransport = (mode: StreamTransport, interval: PollInterval) => {
        if (mode === 'poll') {
            const poll = new BrowserPollStream(makeHandlers(false), interval);
            pollRef.current = poll;
            activeRef.current = poll;
            poll.attachImage(imgRef.current);
            poll.connect();
        } else {
            const ws = new BrowserStream(makeHandlers(true));
            streamRef.current = ws;
            activeRef.current = ws;
            ws.attachImage(imgRef.current);
            ws.connect();
        }
        if (stageRef.current && inputEnabled.current) gesturesRef.current?.attach(stageRef.current, imgRef.current);
    };

    const stopTransport = () => {
        streamRef.current?.dispose();
        pollRef.current?.dispose();
        streamRef.current = null;
        pollRef.current = null;
        activeRef.current = null;
    };

    // The transport effect owns the whole lifecycle, first mount included: one
    // code path builds the stream for whichever transport is current.
    const mounted = useRef(false);
    useEffect(() => {
        if (mounted.current) {
            stopTransport();
            setConnection('connecting');
        }
        mounted.current = true;
        startTransport(transport, pollInterval);
        return () => {
            stopTransport();
        };
        // pollInterval is read at build time; changing it rebuilds via the menu
        // callback, not this effect.
        // eslint-disable-next-line react-hooks/exhaustive-deps
    }, [transport]);

    // Latest choice for the visibility handlers without re-running the effect.
    const choiceRef = useRef({ transport, pollInterval });
    choiceRef.current = { transport, pollInterval };

    useEffect(() => {
        const host: GestureHost = {
            tap: (x, y) => activeRef.current?.send({ type: 'tap', x, y, button: 1 }),
            down: (x, y) => activeRef.current?.send({ type: 'down', x, y, button: 1 }),
            move: (x, y) => activeRef.current?.send({ type: 'move', x, y, button: 1 }),
            up: (x, y) => activeRef.current?.send({ type: 'up', x, y, button: 1 }),
            wheel: (x, y, dx, dy) => activeRef.current?.send({ type: 'wheel', x, y, delta_x: dx, delta_y: dy }),
            onDragChange: setDragging,
        };
        const gestures = new BrowserGestures(host);
        gesturesRef.current = gestures;
        if (cursorRef.current) gestures.attachCursor(cursorRef.current);

        // A hidden page stops being a viewer: closing the socket is the only
        // thing that stops capture on the host, so a backgrounded tab must not
        // keep taking screenshots. iOS may skip visibilitychange on lock, so
        // pagehide/pageshow are handled too.
        const onVisibility = () => {
            if (document.hidden) stopTransport();
            else if (!activeRef.current) startTransport(choiceRef.current.transport, choiceRef.current.pollInterval);
        };
        const onHide = () => stopTransport();
        const onShow = () => {
            if (!document.hidden && !activeRef.current) startTransport(choiceRef.current.transport, choiceRef.current.pollInterval);
        };
        document.addEventListener('visibilitychange', onVisibility);
        window.addEventListener('pagehide', onHide);
        window.addEventListener('pageshow', onShow);
        if (document.hidden) stopTransport();

        return () => {
            document.removeEventListener('visibilitychange', onVisibility);
            window.removeEventListener('pagehide', onHide);
            window.removeEventListener('pageshow', onShow);
            gestures.detach();
            gesturesRef.current = null;
        };
        // Mount-only: gestures live for the page's lifetime; transports swap
        // underneath them through activeRef.
        // eslint-disable-next-line react-hooks/exhaustive-deps
    }, []);

    // Cast can stall (a buried window, a compositor that stops producing) while
    // shot polling keeps working. One automatic switch beats a frozen picture.
    useEffect(() => {
        if (fallbackDone.current) return;
        if (!status || status.mode !== 'cast') return;
        if (connection !== 'open' || !status.capture.active || status.capture.frames === 0) return;
        const timer = window.setInterval(() => {
            const stream = streamRef.current;
            if (!stream || fallbackDone.current) return;
            if (stream.frameAge <= 2500) return;
            fallbackDone.current = true;
            stream.setMode('shot');
            setFallbackNote('Cast stalled — switched to shot mode.');
            window.clearInterval(timer);
        }, 1000);
        return () => window.clearInterval(timer);
    }, [status, connection]);

    useEffect(() => {
        if (!error) return;
        const timer = window.setTimeout(() => setError(null), 6000);
        return () => window.clearTimeout(timer);
    }, [error]);

    const attachStage = (el: HTMLDivElement | null) => {
        stageRef.current = el;
        if (!el) {
            gesturesRef.current?.detach();
            return;
        }
        if (inputEnabled.current) gesturesRef.current?.attach(el, imgRef.current);
    };

    const attachImage = (el: HTMLImageElement | null) => {
        imgRef.current = el;
        streamRef.current?.attachImage(el);
        pollRef.current?.attachImage(el);
        if (stageRef.current && inputEnabled.current) gesturesRef.current?.attach(stageRef.current, el);
    };

    /**
     * Turns page input on for the CTRL lens and off again for VIEW.
     *
     * Listening is opt-in rather than "always attached, ignore events": a stray
     * finger in VIEW must not even reach the page, and while watching, every
     * touch belongs to the zoom pad.
     */
    const setInput = (enabled: boolean) => {
        inputEnabled.current = enabled;
        if (enabled && stageRef.current) gesturesRef.current?.attach(stageRef.current, imgRef.current);
        else gesturesRef.current?.detach();
    };

    const attachCursor = (el: HTMLDivElement | null) => {
        cursorRef.current = el;
        gesturesRef.current?.attachCursor(el);
    };

    const setPollInterval = (ms: PollInterval) => {
        setPollIntervalState(ms);
        if (transport === 'poll') {
            // Rebuild so the running loop picks the new cadence immediately.
            stopTransport();
            setConnection('connecting');
            startTransport('poll', ms);
        }
        persist(transport, ms);
    };

    const switchTransport = (next: StreamTransport) => {
        if (next === transport) return;
        setProbe(null);
        persist(next, pollInterval);
        setTransport(next);
    };

    const retryWs = () => {
        setProbe(null);
        streamRef.current?.resetProbe();
        streamRef.current?.connect();
    };

    return {
        transport,
        probe,
        pollInterval,
        switchTransport,
        retryWs,
        setPollInterval,
        status,
        connection,
        error,
        dragging,
        fallbackNote,
        dismissFallbackNote: () => setFallbackNote(null),
        clearError: () => setError(null),
        attachStage,
        attachImage,
        attachCursor,
        setInput,
        send: (msg: ClientMessage) => activeRef.current?.send(msg),
        navigate: (url: string) => activeRef.current?.navigate(url),
        setPreset: (preset: string) => activeRef.current?.setPreset(preset),
        setMode: (mode: BrowserMode) => activeRef.current?.setMode(mode),
        refresh: () => activeRef.current?.refresh(),
    };
}
