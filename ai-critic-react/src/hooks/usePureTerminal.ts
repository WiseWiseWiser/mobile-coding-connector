import { useEffect, useRef, useState } from 'react';
import type { MutableRefObject } from 'react';
import { Terminal as XTerm } from '@xterm/xterm';
import { FitAddon } from '@xterm/addon-fit';
import type { TerminalTheme } from '../types/terminal';
import { useCurrent } from './useCurrent';
import { TerminalConnection, type TerminalConnectionState } from './terminalConnection';

export type { TerminalTheme, TerminalConnectionState };

export interface PureTerminalOptions {
    theme?: TerminalTheme;
    fontSize?: number;
    fontFamily?: string;
    cwd?: string;
    name?: string;
    initialCommand?: string;
    sessionId?: string;
    onSessionId?: (sessionId: string) => void;
    onCloseRequest?: () => void;
    onConnectionChange?: (connected: boolean) => void;
    onStateChange?: (state: TerminalConnectionState) => void;
    onCtrlModeConsumed?: () => void;
}

export interface PureTerminalReturn {
    terminalRef: MutableRefObject<HTMLDivElement | null>;
    connected: boolean;
    state: TerminalConnectionState;
    sendKey: (key: string) => void;
    focus: () => void;
    reconnect: () => void;
    fit: () => void;
    ctrlModeRef: MutableRefObject<boolean>;
}

const defaultTheme: TerminalTheme = {
    background: '#0f172a', foreground: '#e2e8f0', cursor: '#60a5fa',
    cursorAccent: '#0f172a', selectionBackground: '#334155',
    black: '#0f172a', red: '#ef4444', green: '#22c55e', yellow: '#eab308',
    blue: '#3b82f6', magenta: '#a855f7', cyan: '#06b6d4', white: '#f1f5f9',
    brightBlack: '#475569', brightRed: '#f87171', brightGreen: '#4ade80',
    brightYellow: '#facc15', brightBlue: '#60a5fa', brightMagenta: '#c084fc',
    brightCyan: '#22d3ee', brightWhite: '#ffffff',
};

/** Renderer lifetime is separate from recoverable WebSocket attachments. */
export function usePureTerminal(options: PureTerminalOptions = {}): PureTerminalReturn {
    const [state, setState] = useState<TerminalConnectionState>('connecting');
    const terminalRef = useRef<HTMLDivElement | null>(null);
    const xtermRef = useRef<XTerm | null>(null);
    const fitAddonRef = useRef<FitAddon | null>(null);
    const connectionRef = useRef<TerminalConnection | null>(null);
    const ctrlModeRef = useRef(false);
    const optionsRef = useCurrent(options);

    const fit = () => {
        const element = terminalRef.current;
        if (!element || element.clientWidth === 0 || element.clientHeight === 0) return;
        const addon = fitAddonRef.current;
        if (!addon) return;
        addon.fit();
        const dimensions = addon.proposeDimensions();
        if (dimensions && dimensions.cols > 0 && dimensions.rows > 0) {
            connectionRef.current?.send(JSON.stringify({ type: 'resize', ...dimensions }));
        }
    };
    const fitRef = useCurrent(fit);

    useEffect(() => {
        const element = terminalRef.current;
        if (!element) return;
        const opts = optionsRef.current;
        const xterm = new XTerm({
            cursorBlink: true, fontSize: opts.fontSize ?? 14,
            fontFamily: opts.fontFamily ?? 'Monaco, Menlo, "Ubuntu Mono", Consolas, monospace',
            theme: opts.theme ?? defaultTheme, allowProposedApi: true, scrollback: 10000,
        });
        const addon = new FitAddon();
        xterm.loadAddon(addon);
        xterm.open(element);
        xtermRef.current = xterm;
        fitAddonRef.current = addon;
        const connection = new TerminalConnection({
            sessionId: opts.sessionId,
            initialCommand: opts.initialCommand,
            url: sessionId => {
                const params = new URLSearchParams({ attach_mode: 'web' });
                if (sessionId) params.set('session_id', sessionId);
                if (opts.cwd) params.set('cwd', opts.cwd);
                if (opts.name) params.set('name', opts.name);
                const protocol = location.protocol === 'https:' ? 'wss:' : 'ws:';
                return `${protocol}//${location.host}/api/terminal?${params}`;
            },
            onSessionId: id => optionsRef.current.onSessionId?.(id),
            onOutput: data => xterm.write(data),
            onReplayStart: () => xterm.reset(),
            onState: next => {
                setState(next);
                optionsRef.current.onConnectionChange?.(next === 'connected');
                optionsRef.current.onStateChange?.(next);
                if (next === 'connected') scheduleFit();
            },
        });
        connectionRef.current = connection;
        // Defer opening so React StrictMode's setup/cleanup probe cannot create
        // an orphan durable shell before the real mount.
        const startTimer = window.setTimeout(() => connection.start(), 0);
        let resizeFrame: number | null = null;
        function scheduleFit() {
            if (resizeFrame !== null) cancelAnimationFrame(resizeFrame);
            resizeFrame = requestAnimationFrame(() => {
                resizeFrame = null;
                fitRef.current();
            });
        }
        const dataDisposable = xterm.onData(data => {
            if (ctrlModeRef.current && data.length === 1) {
                ctrlModeRef.current = false;
                optionsRef.current.onCtrlModeConsumed?.();
                const code = data.toUpperCase().charCodeAt(0);
                if (code >= 65 && code <= 90) data = String.fromCharCode(code - 64);
            }
            connection.send(data);
        });
        const recover = () => {
            if (!document.hidden && navigator.onLine !== false) connection.reconnect();
        };
        const onVisibility = () => { if (!document.hidden) recover(); };
        document.addEventListener('visibilitychange', onVisibility);
        window.addEventListener('pageshow', recover);
        window.addEventListener('online', recover);
        window.addEventListener('resize', scheduleFit);
        const resizeObserver = new ResizeObserver(scheduleFit);
        resizeObserver.observe(element);
        const intersectionObserver = new IntersectionObserver(entries => {
            if (entries[0]?.isIntersecting) scheduleFit();
        });
        intersectionObserver.observe(element);

        // xterm captures native touch scrolling on iOS.
        const viewport = element.querySelector('.xterm-viewport') as HTMLElement | null;
        let touchY = 0;
        let scrollTop = 0;
        const touchStart = (e: TouchEvent) => {
            if (e.touches.length !== 1) return;
            touchY = e.touches[0].clientY;
            scrollTop = viewport?.scrollTop ?? 0;
        };
        const touchMove = (e: TouchEvent) => {
            if (e.touches.length === 1 && viewport) {
                const delta = touchY - e.touches[0].clientY;
                if (Math.abs(delta) > 5) viewport.scrollTop = scrollTop + delta;
            }
        };
        viewport?.addEventListener('touchstart', touchStart, { passive: true });
        viewport?.addEventListener('touchmove', touchMove, { passive: true });
        scheduleFit();
        return () => {
            clearTimeout(startTimer);
            if (resizeFrame !== null) cancelAnimationFrame(resizeFrame);
            document.removeEventListener('visibilitychange', onVisibility);
            window.removeEventListener('pageshow', recover);
            window.removeEventListener('online', recover);
            window.removeEventListener('resize', scheduleFit);
            resizeObserver.disconnect();
            intersectionObserver.disconnect();
            viewport?.removeEventListener('touchstart', touchStart);
            viewport?.removeEventListener('touchmove', touchMove);
            dataDisposable.dispose();
            connection.dispose();
            connectionRef.current = null;
            xterm.dispose();
            xtermRef.current = null;
            fitAddonRef.current = null;
        };
    }, [fitRef, optionsRef]);

    return {
        terminalRef, connected: state === 'connected', state, ctrlModeRef,
        sendKey: key => { connectionRef.current?.send(key); xtermRef.current?.focus(); },
        focus: () => xtermRef.current?.focus(),
        reconnect: () => connectionRef.current?.reconnect(),
        fit,
    };
}
