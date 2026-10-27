import { useEffect, useRef, useState } from 'react';
import { BrowserStream } from './browserStream';
import { BrowserGestures } from './browserGestures';
import type { GestureHost } from './browserGestures';
import type { BrowserMode, BrowserStatus, ConnectionState } from './browserWire';

/**
 * Owns the socket and the gesture translator for the lifetime of the page.
 *
 * Unlike /screen there is no control handshake: every viewer may click. The
 * browser window is a shared surface, and a second viewer taking over mid-click
 * is less surprising than a modal nobody can dismiss on a phone.
 */
export function useBrowserStream() {
    const streamRef = useRef<BrowserStream | null>(null);
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

    useEffect(() => {
        const host: GestureHost = {
            tap: (x, y) => streamRef.current?.send({ type: 'tap', x, y, button: 1 }),
            down: (x, y) => streamRef.current?.send({ type: 'down', x, y, button: 1 }),
            move: (x, y) => streamRef.current?.send({ type: 'move', x, y, button: 1 }),
            up: (x, y) => streamRef.current?.send({ type: 'up', x, y, button: 1 }),
            wheel: (x, y, dx, dy) => streamRef.current?.send({ type: 'wheel', x, y, delta_x: dx, delta_y: dy }),
            onDragChange: setDragging,
        };
        const stream = new BrowserStream({
            onStatus: setStatus,
            onConnection: setConnection,
            onError: (message) => setError(message),
        });
        const gestures = new BrowserGestures(host);
        streamRef.current = stream;
        gesturesRef.current = gestures;
        // The image ref landed before this effect ran, so a stream created here
        // has to pick up the element it paints into.
        stream.attachImage(imgRef.current);
        stream.connect();
        if (cursorRef.current) gestures.attachCursor(cursorRef.current);

        // A hidden page stops being a viewer: closing the socket is the only
        // thing that stops capture on the host, so a backgrounded tab must not
        // keep taking screenshots. iOS may skip visibilitychange on lock, so
        // pagehide/pageshow are handled too.
        const onVisibility = () => {
            if (document.hidden) stream.pause();
            else stream.resume();
        };
        const onHide = () => stream.pause();
        const onShow = () => {
            if (!document.hidden) stream.resume();
        };
        document.addEventListener('visibilitychange', onVisibility);
        window.addEventListener('pagehide', onHide);
        window.addEventListener('pageshow', onShow);
        if (document.hidden) stream.pause();

        return () => {
            document.removeEventListener('visibilitychange', onVisibility);
            window.removeEventListener('pagehide', onHide);
            window.removeEventListener('pageshow', onShow);
            gestures.detach();
            stream.dispose();
            streamRef.current = null;
            gesturesRef.current = null;
        };
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

    return {
        stream: streamRef,
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
        send: (msg: Parameters<BrowserStream['send']>[0]) => streamRef.current?.send(msg),
        navigate: (url: string) => streamRef.current?.navigate(url),
        setPreset: (preset: string) => streamRef.current?.setPreset(preset),
        setMode: (mode: BrowserMode) => streamRef.current?.setMode(mode),
        refresh: () => streamRef.current?.refresh(),
    };
}
