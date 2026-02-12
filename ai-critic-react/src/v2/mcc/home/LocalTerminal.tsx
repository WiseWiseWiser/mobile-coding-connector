import { useEffect, useRef, useState } from 'react';
import { Terminal as XTerm } from '@xterm/xterm';
import { FitAddon } from '@xterm/addon-fit';
import '@xterm/xterm/css/xterm.css';
import './LocalTerminal.css';

interface LocalTerminalProps {
    cwd?: string;
    onClose: () => void;
}

// Mobile-friendly dark theme matching the main terminal
const v2Theme = {
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

export function LocalTerminal({ cwd, onClose }: LocalTerminalProps) {
    const terminalRef = useRef<HTMLDivElement>(null);
    const xtermRef = useRef<XTerm | null>(null);
    const fitAddonRef = useRef<FitAddon | null>(null);
    const wsRef = useRef<WebSocket | null>(null);
    const [connected, setConnected] = useState(false);
    const [connecting, setConnecting] = useState(true);
    const [error, setError] = useState('');

    useEffect(() => {
        let disposed = false;

        if (!terminalRef.current) return;

        // Setup xterm
        const xterm = new XTerm({
            cursorBlink: true,
            fontSize: 14,
            fontFamily: 'Monaco, Menlo, "Ubuntu Mono", Consolas, monospace',
            theme: v2Theme,
            allowProposedApi: true,
            scrollback: 10000,
        });
        const fitAddon = new FitAddon();
        xterm.loadAddon(fitAddon);
        xterm.open(terminalRef.current);
        fitAddon.fit();

        xtermRef.current = xterm;
        fitAddonRef.current = fitAddon;

        // Mobile touch scroll support
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

        // Setup WebSocket for local terminal
        const protocol = window.location.protocol === 'https:' ? 'wss:' : 'ws:';
        const params = new URLSearchParams();
        if (cwd) params.set('cwd', cwd);
        params.set('name', 'File Manager');
        const wsUrl = `${protocol}//${window.location.host}/api/terminal${params.toString() ? '?' + params.toString() : ''}`;
        
        const ws = new WebSocket(wsUrl);
        ws.binaryType = 'arraybuffer';
        wsRef.current = ws;

        const sendResize = () => {
            if (ws.readyState !== WebSocket.OPEN || !fitAddon) return;
            const dimensions = fitAddon.proposeDimensions();
            if (dimensions) {
                ws.send(JSON.stringify({ type: 'resize', cols: dimensions.cols, rows: dimensions.rows }));
            }
        };

        ws.onopen = () => {
            if (disposed) return;
            setConnected(true);
            setConnecting(false);
            setTimeout(() => {
                if (disposed) return;
                fitAddon.fit();
                sendResize();
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
                    if (msg.type === 'error') {
                        setError(msg.message || 'Connection error');
                    }
                } catch {
                    xterm.write(text);
                }
            }
        };

        ws.onclose = () => {
            if (disposed) return;
            setConnected(false);
            if (!error) {
                setError('Connection closed');
            }
        };

        ws.onerror = () => {
            if (disposed) return;
            setError('Connection error');
            setConnecting(false);
        };

        // Handle terminal input
        xterm.onData((data) => {
            if (ws.readyState === WebSocket.OPEN) {
                ws.send(data);
            }
        });

        // Handle resize
        const handleResize = () => {
            if (disposed) return;
            fitAddon.fit();
            sendResize();
        };
        window.addEventListener('resize', handleResize);

        // Cleanup
        return () => {
            disposed = true;
            window.removeEventListener('resize', handleResize);
            if (viewport) {
                viewport.removeEventListener('touchstart', handleTouchStart);
                viewport.removeEventListener('touchmove', handleTouchMove);
                viewport.removeEventListener('touchend', handleTouchEnd);
            }
            ws.close();
            xterm.dispose();
        };
    }, [cwd]);

    // Keyboard shortcuts
    const handleCtrlC = () => {
        if (wsRef.current?.readyState === WebSocket.OPEN) {
            wsRef.current.send('\x03');
        }
    };

    const handleEsc = () => {
        if (wsRef.current?.readyState === WebSocket.OPEN) {
            wsRef.current.send('\x1b');
        }
    };

    const handleCtrlD = () => {
        if (wsRef.current?.readyState === WebSocket.OPEN) {
            wsRef.current.send('\x04');
        }
    };

    const handleTab = () => {
        if (wsRef.current?.readyState === WebSocket.OPEN) {
            wsRef.current.send('\t');
        }
    };

    const handlePaste = async () => {
        try {
            const text = await navigator.clipboard.readText();
            if (text && wsRef.current?.readyState === WebSocket.OPEN) {
                wsRef.current.send(text);
            }
        } catch {
            // Ignore paste errors
        }
    };

    return (
        <div className="local-terminal">
            <div className="local-terminal-header">
                <div className="local-terminal-info">
                    <span className="local-terminal-title">
                        {cwd || 'Terminal'}
                    </span>
                    {connecting && <span className="local-terminal-status connecting">Connecting...</span>}
                    {connected && <span className="local-terminal-status connected">Connected</span>}
                    {error && !connected && <span className="local-terminal-status error">{error}</span>}
                </div>
                <button className="local-terminal-close-btn" onClick={onClose}>Close</button>
            </div>
            <div className="local-terminal-content" ref={terminalRef} />
            <div className="local-terminal-actions">
                <button className="local-terminal-btn" onClick={handleTab}>Tab</button>
                <button className="local-terminal-btn" onClick={handleEsc}>Esc</button>
                <button className="local-terminal-btn" onClick={handleCtrlC}>^C</button>
                <button className="local-terminal-btn" onClick={handleCtrlD}>^D</button>
                <button className="local-terminal-btn" onClick={handlePaste}>Paste</button>
            </div>
        </div>
    );
}
