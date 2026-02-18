import { useEffect, useRef, useState, useCallback } from 'react';
import type { MutableRefObject } from 'react';
import { Terminal as XTerm } from '@xterm/xterm';
import { FitAddon } from '@xterm/addon-fit';
import type { TerminalTheme } from '../types/terminal';
import { useCurrent } from './useCurrent';

export type { TerminalTheme };

export interface PureTerminalOptions {
    theme?: TerminalTheme;
    fontSize?: number;
    fontFamily?: string;
    /** Working directory for the terminal session */
    cwd?: string;
    /** Display name for the terminal session */
    name?: string;
    /** Command to run automatically after the terminal connects (only for new sessions) */
    initialCommand?: string;
    /** Existing session ID to reconnect to. If provided, reconnects instead of creating a new session */
    sessionId?: string;
    /** Called when the backend assigns a session ID (for new sessions) */
    onSessionId?: (sessionId: string) => void;
    /** Called when the user presses any key after connection is closed, requesting cleanup */
    onCloseRequest?: () => void;
    /** Called when connection status changes */
    onConnectionChange?: (connected: boolean) => void;
    /** Called when Ctrl mode is consumed by a keystroke */
    onCtrlModeConsumed?: () => void;
}

export interface PureTerminalReturn {
    terminalRef: MutableRefObject<HTMLDivElement | null>;
    connected: boolean;
    sendKey: (key: string) => void;
    focus: () => void;
    reconnect: () => void;
    /** Refit the terminal to its container. Call this when the terminal becomes visible. */
    fit: () => void;
    /** Ref to control Ctrl modifier mode. When true, next key input is converted to a control character. */
    ctrlModeRef: MutableRefObject<boolean>;
}

const defaultTheme: TerminalTheme = {
    background: '#0f172a',
    foreground: '#e2e8f0',
    cursor: '#60a5fa',
    cursorAccent: '#0f172a',
    selectionBackground: '#334155',
    black: '#0f172a',
    red: '#ef4444',
    green: '#22c55e',
    yellow: '#eab308',
    blue: '#3b82f6',
    magenta: '#a855f7',
    cyan: '#06b6d4',
    white: '#f1f5f9',
    brightBlack: '#475569',
    brightRed: '#f87171',
    brightGreen: '#4ade80',
    brightYellow: '#facc15',
    brightBlue: '#60a5fa',
    brightMagenta: '#c084fc',
    brightCyan: '#22d3ee',
    brightWhite: '#ffffff',
};

/**
 * Pure terminal hook - manages xterm.js instance and WebSocket connection.
 * This is the core terminal logic without any UI concerns (tabs, quick input, etc.)
 */
export function usePureTerminal(options: PureTerminalOptions = {}): PureTerminalReturn {
    const [connected, setConnected] = useState(false);
    const terminalRef = useRef<HTMLDivElement | null>(null);
    const xtermRef = useRef<XTerm | null>(null);
    const fitAddonRef = useRef<FitAddon | null>(null);
    const wsRef = useRef<WebSocket | null>(null);
    const cleanupRef = useRef<(() => void) | null>(null);
    const ctrlModeRef = useRef(false);

    // Store options in a ref so setup reads them imperatively
    const optionsRef = useCurrent(options);

    const setupRef = useRef<(() => void) | null>(null);

    // Imperative setup function — creates xterm + WebSocket, stores cleanup in ref
    setupRef.current = () => {
        // Tear down previous instance if any
        cleanupRef.current?.();
        cleanupRef.current = null;

        if (!terminalRef.current) return;

        let disposed = false;
        let hadUserInput = false;
        const {
            theme = defaultTheme,
            fontSize = 14,
            fontFamily = 'Monaco, Menlo, "Ubuntu Mono", Consolas, monospace',
            cwd,
            name,
            initialCommand,
            sessionId,
        } = optionsRef.current;

        // ---- xterm setup ----
        const xterm = new XTerm({
            cursorBlink: true,
            fontSize,
            fontFamily,
            theme,
            allowProposedApi: true,
            scrollback: 10000,
        });
        const fitAddon = new FitAddon();
        xterm.loadAddon(fitAddon);
        xterm.open(terminalRef.current);
        fitAddon.fit();

        xtermRef.current = xterm;
        fitAddonRef.current = fitAddon;

        // ---- Mobile touch scroll support ----
        const viewport = terminalRef.current.querySelector('.xterm-viewport') as HTMLElement | null;
        let touchStartY = 0;
        let touchScrollTop = 0;
        let isTouchScrolling = false;

        const handleTouchStart = (e: TouchEvent) => {
            if (e.touches.length !== 1) return;
            touchStartY = e.touches[0].clientY;
            touchScrollTop = viewport?.scrollTop ?? 0;
            isTouchScrolling = false;
        };
        const handleTouchMove = (e: TouchEvent) => {
            if (e.touches.length !== 1 || !viewport) return;
            const deltaY = touchStartY - e.touches[0].clientY;
            if (!isTouchScrolling && Math.abs(deltaY) > 5) {
                isTouchScrolling = true;
            }
            if (isTouchScrolling) {
                viewport.scrollTop = touchScrollTop + deltaY;
            }
        };
        const handleTouchEnd = () => {
            isTouchScrolling = false;
        };

        if (viewport) {
            viewport.addEventListener('touchstart', handleTouchStart, { passive: true });
            viewport.addEventListener('touchmove', handleTouchMove, { passive: true });
            viewport.addEventListener('touchend', handleTouchEnd, { passive: true });
        }

        // ---- WebSocket setup ----
        const protocol = window.location.protocol === 'https:' ? 'wss:' : 'ws:';
        const params = new URLSearchParams();
        if (sessionId) params.set('session_id', sessionId);
        if (cwd) params.set('cwd', cwd);
        if (name) params.set('name', name);
        const queryString = params.toString();
        const wsUrl = `${protocol}//${window.location.host}/api/terminal${queryString ? '?' + queryString : ''}`;
        const ws = new WebSocket(wsUrl);
        ws.binaryType = 'arraybuffer';
        wsRef.current = ws;

        const isReconnecting = !!sessionId;

        const sendResize = () => {
            if (ws.readyState !== WebSocket.OPEN) return;
            const dimensions = fitAddon.proposeDimensions();
            if (dimensions) {
                ws.send(JSON.stringify({ type: 'resize', cols: dimensions.cols, rows: dimensions.rows }));
            }
        };

        ws.onopen = () => {
            if (disposed) return;
            setConnected(true);
            optionsRef.current.onConnectionChange?.(true);
            setTimeout(() => {
                if (disposed) return;
                fitAddon.fit();
                sendResize();
                if (!isReconnecting && initialCommand) {
                    setTimeout(() => {
                        if (disposed) return;
                        ws.send(initialCommand + '\n');
                    }, 300);
                }
            }, 100);
        };

        ws.onmessage = (event) => {
            if (disposed) return;
            if (event.data instanceof ArrayBuffer) {
                xterm.write(new TextDecoder().decode(event.data));
            } else {
                const text = event.data as string;
                try {
                    const msg = JSON.parse(text);
                    if (msg.type === 'session_id' && msg.session_id) {
                        optionsRef.current.onSessionId?.(msg.session_id);
                        return;
                    }
                } catch {
                    // Not JSON — write as terminal output
                }
                xterm.write(text);
            }
        };

        ws.onerror = () => {
            if (disposed) return;
            xterm.writeln('\r\n\x1b[31mConnection error\x1b[0m');
            setConnected(false);
            optionsRef.current.onConnectionChange?.(false);
        };

        ws.onclose = () => {
            if (disposed) return;
            xterm.writeln('\r\n\x1b[33mConnection closed\x1b[0m');
            xterm.writeln('\x1b[90m[Press any key to close]\x1b[0m');
            setConnected(false);
            optionsRef.current.onConnectionChange?.(false);

            // Listen for any key press to request cleanup
            const keyDisposable = xterm.onKey(() => {
                keyDisposable.dispose();
                optionsRef.current.onCloseRequest?.();
            });
        };

        xterm.onData((data) => {
            if (ws.readyState !== WebSocket.OPEN) return;
            hadUserInput = true;

            // When Ctrl mode is active, convert next single printable char to control character
            if (ctrlModeRef.current) {
                if (data.length === 1) {
                    ctrlModeRef.current = false;
                    optionsRef.current.onCtrlModeConsumed?.();
                    const charCode = data.charCodeAt(0);
                    if (charCode >= 97 && charCode <= 122) {
                        ws.send(String.fromCharCode(charCode - 96));
                        return;
                    }
                    if (charCode >= 65 && charCode <= 90) {
                        ws.send(String.fromCharCode(charCode - 64));
                        return;
                    }
                }
            }
            ws.send(data);
        });

        // ---- Resize handler ----
        const handleResize = () => {
            fitAddonRef.current?.fit();
            sendResize();
        };
        window.addEventListener('resize', handleResize);

        // ---- Visibility observer ----
        let intersectionObserver: IntersectionObserver | null = null;
        if (terminalRef.current) {
            intersectionObserver = new IntersectionObserver(
                (entries) => {
                    const entry = entries[0];
                    if (entry?.isIntersecting && ws.readyState === WebSocket.OPEN) {
                        setTimeout(() => {
                            if (disposed) return;
                            fitAddon.fit();
                            sendResize();
                        }, 50);
                    }
                },
                { threshold: 0.1 }
            );
            intersectionObserver.observe(terminalRef.current);
        }

        xterm.focus();

        // Store cleanup
        cleanupRef.current = () => {
            disposed = true;
            window.removeEventListener('resize', handleResize);
            intersectionObserver?.disconnect();
            if (viewport) {
                viewport.removeEventListener('touchstart', handleTouchStart);
                viewport.removeEventListener('touchmove', handleTouchMove);
                viewport.removeEventListener('touchend', handleTouchEnd);
            }
            const shouldDelete = !sessionId && !hadUserInput;
            if (ws.readyState === WebSocket.OPEN) {
                ws.close(shouldDelete ? 4000 : 1000);
            } else if (ws.readyState === WebSocket.CONNECTING) {
                ws.onopen = () => {
                    ws.close(shouldDelete ? 4000 : 1000);
                };
                ws.onerror = () => {
                    ws.close();
                };
            }
            wsRef.current = null;
            xterm.dispose();
            xtermRef.current = null;
            fitAddonRef.current = null;
        };
    };

    // Run setup once on mount, clean up on unmount
    useEffect(() => {
        setupRef.current?.();
        return () => {
            cleanupRef.current?.();
            cleanupRef.current = null;
        };
    }, []);

    const sendKey = useCallback((key: string) => {
        if (wsRef.current?.readyState === WebSocket.OPEN) {
            wsRef.current.send(key);
            xtermRef.current?.focus();
        }
    }, []);

    const focus = useCallback(() => {
        xtermRef.current?.focus();
    }, []);

    const reconnect = useCallback(() => {
        setupRef.current?.();
    }, []);

    const fit = useCallback(() => {
        if (fitAddonRef.current && wsRef.current?.readyState === WebSocket.OPEN) {
            fitAddonRef.current.fit();
            const dimensions = fitAddonRef.current.proposeDimensions();
            if (dimensions) {
                wsRef.current.send(JSON.stringify({ type: 'resize', cols: dimensions.cols, rows: dimensions.rows }));
            }
        }
    }, []);

    return { terminalRef, connected, sendKey, focus, reconnect, fit, ctrlModeRef };
}
