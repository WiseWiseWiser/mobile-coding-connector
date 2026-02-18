import { useEffect, useRef, useState, useCallback } from 'react';
import type { MutableRefObject } from 'react';
import { Terminal as XTerm } from '@xterm/xterm';
import { FitAddon } from '@xterm/addon-fit';
import { getFakeShellServer, type FakeShellSession } from './FakeShellServer';
import type { TerminalTheme } from '../../hooks/usePureTerminal';

export type { TerminalTheme };

export interface UseFakeTerminalOptions {
    theme?: TerminalTheme;
    fontSize?: number;
    fontFamily?: string;
    cwd?: string;
    name?: string;
    initialCommand?: string;
}

export interface UseFakeTerminalReturn {
    terminalRef: MutableRefObject<HTMLDivElement | null>;
    connected: boolean;
    sendKey: (key: string) => void;
    focus: () => void;
    reconnect: () => void;
    fit: () => void;
    resetToFit: () => void;
    cols: number;
    rows: number;
    setDimensions: (cols: number, rows: number) => void;
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
 * useFakeTerminal - A hook that connects to the FakeShellServer instead of a real WebSocket.
 * This provides a fully interactive terminal experience running entirely in the browser.
 */
export function useFakeTerminal(options: UseFakeTerminalOptions = {}): UseFakeTerminalReturn {
    const [connected, setConnected] = useState(false);
    const [dimensions, setDimensionsState] = useState({ cols: 80, rows: 24 });
    const terminalRef = useRef<HTMLDivElement | null>(null);
    const xtermRef = useRef<XTerm | null>(null);
    const fitAddonRef = useRef<FitAddon | null>(null);
    const sessionRef = useRef<FakeShellSession | null>(null);
    const cleanupRef = useRef<(() => void) | null>(null);
    const dimensionsRef = useRef(dimensions);
    const manualResizeRef = useRef(false);

    const setup = useCallback(() => {
        // Tear down previous instance if any
        cleanupRef.current?.();
        cleanupRef.current = null;

        if (!terminalRef.current) return;

        let disposed = false;
        const {
            theme = defaultTheme,
            fontSize = 14,
            fontFamily = 'Monaco, Menlo, "Ubuntu Mono", Consolas, monospace',
            cwd,
            name,
            initialCommand,
        } = options;

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

        // ---- Connect to FakeShellServer ----
        const server = getFakeShellServer();
        const session = server.createSession({ cwd, name });
        sessionRef.current = session;

        // Handle incoming data from fake shell
        const unsubscribeData = session.onData((data) => {
            if (disposed) return;
            xterm.write(data);
        });

        // Handle session close
        const unsubscribeClose = session.onClose(() => {
            if (disposed) return;
            setConnected(false);
            xterm.writeln('\r\n\x1b[33mSession closed\x1b[0m');
        });

        // Mark as connected
        setConnected(true);

        // Handle terminal input
        xterm.onData((data) => {
            if (disposed) return;
            session.send(data);
        });

        // Send initial command if provided
        if (initialCommand) {
            setTimeout(() => {
                if (!disposed) {
                    session.send(initialCommand + '\r');
                }
            }, 500);
        }

        // ---- Resize handler ----
        const handleResize = () => {
            if (!manualResizeRef.current) {
                fitAddonRef.current?.fit();
            } else {
                if (xtermRef.current && terminalRef.current) {
                    const dims = dimensionsRef.current;
                    xtermRef.current.resize(dims.cols, dims.rows);
                }
            }
        };
        window.addEventListener('resize', handleResize);

        // ---- Visibility observer ----
        let intersectionObserver: IntersectionObserver | null = null;
        if (terminalRef.current) {
            intersectionObserver = new IntersectionObserver(
                (entries) => {
                    const entry = entries[0];
                    if (entry?.isIntersecting) {
                        setTimeout(() => {
                            if (disposed) return;
                            if (!manualResizeRef.current) {
                                fitAddon.fit();
                            }
                        }, 50);
                    }
                },
                { threshold: 0.1 }
            );
            intersectionObserver.observe(terminalRef.current);
        }

        // Initial fit if not manually resized
        if (!manualResizeRef.current) {
            setTimeout(() => {
                if (!disposed) {
                    fitAddon.fit();
                }
            }, 100);
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
            unsubscribeData();
            unsubscribeClose();
            session.close();
            sessionRef.current = null;
            xterm.dispose();
            xtermRef.current = null;
            fitAddonRef.current = null;
        };
    }, []);

    // Run setup once on mount, clean up on unmount
    useEffect(() => {
        setup();
        return () => {
            cleanupRef.current?.();
            cleanupRef.current = null;
        };
    }, [setup]);

    const sendKey = useCallback((key: string) => {
        sessionRef.current?.send(key);
        xtermRef.current?.focus();
    }, []);

    const focus = useCallback(() => {
        xtermRef.current?.focus();
    }, []);

    const reconnect = useCallback(() => {
        setup();
    }, [setup]);

    const fit = useCallback(() => {
        fitAddonRef.current?.fit();
    }, []);

    const resetToFit = useCallback(() => {
        manualResizeRef.current = false;
        if (terminalRef.current) {
            terminalRef.current.style.width = '';
        }
        fitAddonRef.current?.fit();
    }, []);

    const setDimensions = useCallback((cols: number, rows: number) => {
        dimensionsRef.current = { cols, rows };
        setDimensionsState({ cols, rows });
        manualResizeRef.current = true;
        if (xtermRef.current && terminalRef.current) {
            xtermRef.current.resize(cols, rows);
            const charWidth = 8.4;
            const terminalWidth = cols * charWidth;
            terminalRef.current.style.width = `${terminalWidth}px`;
        }
        sessionRef.current?.resize(cols, rows);
    }, []);

    return { terminalRef, connected, sendKey, focus, reconnect, fit, resetToFit, cols: dimensions.cols, rows: dimensions.rows, setDimensions };
}
