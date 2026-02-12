import { useEffect, useRef, useState } from 'react';
import { Terminal as XTerm } from '@xterm/xterm';
import { FitAddon } from '@xterm/addon-fit';
import '@xterm/xterm/css/xterm.css';
import { loadSSHKeys } from './settings/gitStorage';
import { encryptWithServerKey } from './crypto';
import './EmbeddedTerminal.css';

interface EmbeddedTerminalProps {
    host: string;
    port: number;
    username: string;
    sshKeyId: string;
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

export function EmbeddedTerminal({ host, port, username, sshKeyId, onClose }: EmbeddedTerminalProps) {
    const terminalRef = useRef<HTMLDivElement>(null);
    const xtermRef = useRef<XTerm | null>(null);
    const fitAddonRef = useRef<FitAddon | null>(null);
    const wsRef = useRef<WebSocket | null>(null);
    const [connected, setConnected] = useState(false);
    const [connecting, setConnecting] = useState(true);
    const [error, setError] = useState('');
    const [initializing, setInitializing] = useState(true);

    useEffect(() => {
        let disposed = false;

        const initTerminal = async () => {
            try {
                // Load and encrypt SSH key
                const sshKeys = loadSSHKeys();
                const sshKey = sshKeys.find(k => k.id === sshKeyId);
                
                if (!sshKey) {
                    setError('SSH key not found in localStorage');
                    setConnecting(false);
                    setInitializing(false);
                    return;
                }

                // Encrypt the private key
                let encryptedKey: string;
                try {
                    encryptedKey = await encryptWithServerKey(sshKey.privateKey);
                } catch (e) {
                    setError(e instanceof Error ? e.message : 'Failed to encrypt SSH key');
                    setConnecting(false);
                    setInitializing(false);
                    return;
                }

                if (disposed) return;

                // Setup xterm
                if (!terminalRef.current) return;

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

                // Setup WebSocket for SSH connection
                const protocol = window.location.protocol === 'https:' ? 'wss:' : 'ws:';
                const params = new URLSearchParams({
                    ssh: 'true',
                    host,
                    port: port.toString(),
                    user: username,
                });
                const wsUrl = `${protocol}//${window.location.host}/api/terminal?${params.toString()}`;
                
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
                    // Send encrypted SSH key as first message
                    ws.send(JSON.stringify({
                        type: 'ssh_key',
                        key: encryptedKey,
                    }));
                    setConnected(true);
                    setConnecting(false);
                    setInitializing(false);
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
                                setInitializing(false);
                            } else if (msg.type === 'session_id') {
                                // Session established
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
                    setInitializing(false);
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

                // Cleanup function
                return () => {
                    window.removeEventListener('resize', handleResize);
                    if (viewport) {
                        viewport.removeEventListener('touchstart', handleTouchStart);
                        viewport.removeEventListener('touchmove', handleTouchMove);
                        viewport.removeEventListener('touchend', handleTouchEnd);
                    }
                };
            } catch (e) {
                if (!disposed) {
                    setError(e instanceof Error ? e.message : 'Failed to initialize terminal');
                    setConnecting(false);
                    setInitializing(false);
                }
            }
        };

        initTerminal();

        return () => {
            disposed = true;
            wsRef.current?.close();
            xtermRef.current?.dispose();
        };
    }, [host, port, username, sshKeyId]);

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
        <div className="embedded-terminal">
            <div className="embedded-terminal-header">
                <div className="embedded-terminal-info">
                    <span className="embedded-terminal-title">
                        {username}@{host}:{port}
                    </span>
                    {initializing && <span className="embedded-terminal-status connecting">Initializing...</span>}
                    {connecting && !initializing && <span className="embedded-terminal-status connecting">Connecting...</span>}
                    {connected && <span className="embedded-terminal-status connected">Connected</span>}
                    {error && !connected && <span className="embedded-terminal-status error">{error}</span>}
                </div>
                <button className="embedded-terminal-close-btn" onClick={onClose}>Close</button>
            </div>
            <div className="embedded-terminal-content" ref={terminalRef} />
            <div className="embedded-terminal-actions">
                <button className="embedded-terminal-btn" onClick={handleTab}>Tab</button>
                <button className="embedded-terminal-btn" onClick={handleEsc}>Esc</button>
                <button className="embedded-terminal-btn" onClick={handleCtrlC}>^C</button>
                <button className="embedded-terminal-btn" onClick={handleCtrlD}>^D</button>
                <button className="embedded-terminal-btn" onClick={handlePaste}>Paste</button>
            </div>
        </div>
    );
}
