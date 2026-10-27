import { useEffect, useRef, useState } from 'react';
import { RemoteScreen } from './RemoteScreen';
import type { ControlState, ConnectionState } from './RemoteScreen';
import type { Viewport } from './viewport';
import { isInputErrorCode } from './screenInput';
import type { ScreenStatus } from '../../../api/screen';

function viewerLabel(): string {
    const ua = navigator.userAgent;
    if (/iPhone/.test(ua)) return 'iPhone';
    if (/iPad/.test(ua)) return 'iPad';
    if (/Android/.test(ua)) return 'Android';
    if (/Macintosh/.test(ua)) return 'Mac';
    if (/Windows/.test(ua)) return 'Windows';
    return 'Browser';
}

/**
 * Owns one RemoteScreen for the lifetime of the page: connects, exposes the
 * latest status, and hands out a ref callback for the canvas.
 */
export function useScreenSocket() {
    const screenRef = useRef<RemoteScreen | null>(null);
    const [status, setStatus] = useState<ScreenStatus | null>(null);
    const [control, setControl] = useState<ControlState>('view');
    const [connection, setConnection] = useState<ConnectionState>('connecting');
    const [error, setError] = useState<string | null>(null);
    /**
     * Counts input refusals only. A refusal is the one signal that a frame we
     * wrote was dropped, so the composer watches this to put a draft back.
     */
    const [inputError, setInputError] = useState(0);
    const [view, setView] = useState<Viewport>({ zoom: 1, cx: 0, cy: 0 });

    useEffect(() => {
        const screen = new RemoteScreen(viewerLabel(), {
            onStatus: setStatus,
            onControl: setControl,
            onConnection: setConnection,
            onError: (message, code) => {
                setError(message);
                if (isInputErrorCode(code)) setInputError((count) => count + 1);
            },
            onView: setView,
        });
        screenRef.current = screen;
        screen.connect();

        // A hidden page is not a viewer: closing the socket is what tells the
        // server to stop capturing (and to release this page's control), so a
        // backgrounded tab or a locked phone cannot keep the host recording.
        // pagehide/pageshow cover iOS, which may skip visibilitychange around a
        // screen lock; either edge is enough to give up the stream.
        const onVisibility = () => {
            if (document.hidden) screen.pause();
            else screen.resume();
        };
        const onHide = () => screen.pause();
        const onShow = () => {
            if (!document.hidden) screen.resume();
        };
        document.addEventListener('visibilitychange', onVisibility);
        window.addEventListener('pagehide', onHide);
        window.addEventListener('pageshow', onShow);
        // Opening the page in a background tab must not start a capture.
        if (document.hidden) screen.pause();

        return () => {
            document.removeEventListener('visibilitychange', onVisibility);
            window.removeEventListener('pagehide', onHide);
            window.removeEventListener('pageshow', onShow);
            screen.dispose();
            screenRef.current = null;
        };
    }, []);

    const attachCanvas = (canvas: HTMLCanvasElement | null) => {
        const screen = screenRef.current;
        if (!screen) return;
        if (canvas) screen.attach(canvas);
    };

    return {
        screen: screenRef,
        status,
        control,
        connection,
        error,
        inputError,
        view,
        attachCanvas,
        clearError: () => setError(null),
    };
}
