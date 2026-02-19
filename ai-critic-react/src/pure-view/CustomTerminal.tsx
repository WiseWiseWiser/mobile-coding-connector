import { useState, useRef, useEffect, useCallback, forwardRef, useImperativeHandle } from 'react';
import { getFakeShellServer, type FakeShellSession } from '../mockups/fake-shell/FakeShellServer';
import { ShortcutsBar } from './ShortcutsBar';
import './CustomTerminal.css';

export interface CustomTerminalProps {
    className?: string;
    cwd?: string;
    name?: string;
    history: string[];
    wide?: boolean;
    onConnectionChange?: (connected: boolean) => void;
    onCommandExecuted?: (command: string) => void;
}

export interface CustomTerminalHandle {
    sendKey: (key: string) => void;
    focus: () => void;
}

interface TerminalLine {
    id: number;
    content: React.ReactNode;
}

const ANSI_COLORS: Record<string, string> = {
    '0': '#0f172a',
    '1': '#ef4444',
    '2': '#22c55e',
    '3': '#eab308',
    '4': '#3b82f6',
    '5': '#a855f7',
    '6': '#06b6d4',
    '7': '#f1f5f9',
    '30': '#0f172a',
    '31': '#ef4444',
    '32': '#22c55e',
    '33': '#eab308',
    '34': '#3b82f6',
    '35': '#a855f7',
    '36': '#06b6d4',
    '37': '#f1f5f9',
    '90': '#475569',
    '91': '#f87171',
    '92': '#4ade80',
    '93': '#facc15',
    '94': '#60a5fa',
    '95': '#c084fc',
    '96': '#22d3ee',
    '97': '#ffffff',
};

function parseAnsiToHtml(text: string): React.ReactNode {
    const ansiRegex = /\x1b\[([0-9;]*)m/g;
    const parts: { text: string; color?: string; bold: boolean }[] = [];
    let lastIndex = 0;
    let bold = false;
    let currentColor: string | undefined;

    let match;
    while ((match = ansiRegex.exec(text)) !== null) {
        if (match.index > lastIndex) {
            parts.push({
                text: text.slice(lastIndex, match.index),
                color: currentColor,
                bold,
            });
        }

        const codes = match[1].split(';').map(c => parseInt(c) || 0);

        for (const code of codes) {
            if (code === 0) {
                bold = false;
                currentColor = undefined;
            } else if (code === 1) {
                bold = true;
            } else if (code >= 30 && code <= 37) {
                currentColor = ANSI_COLORS[String(code)];
            } else if (code >= 90 && code <= 97) {
                currentColor = ANSI_COLORS[String(code)];
            }
        }

        lastIndex = match.index + match[0].length;
    }

    if (lastIndex < text.length) {
        parts.push({
            text: text.slice(lastIndex),
            color: currentColor,
            bold,
        });
    }

    if (parts.length === 0) {
        return text;
    }

    return parts.map((part, i) => (
        <span
            key={i}
            style={{
                color: part.color,
                fontWeight: part.bold ? 'bold' : undefined,
            }}
        >
            {part.text}
        </span>
    ));
}

function fuzzyMatch(query: string, candidate: string): boolean {
    const q = query.toLowerCase();
    const c = candidate.toLowerCase();
    let qi = 0;
    for (let i = 0; i < c.length && qi < q.length; i++) {
        if (c[i] === q[qi]) qi++;
    }
    return qi === q.length;
}

const LINE_HEIGHT = 19.5;

export const CustomTerminal = forwardRef<CustomTerminalHandle, CustomTerminalProps>(function CustomTerminal({
    className = '',
    cwd = '/home/user',
    name = 'mock-shell',
    history,
    wide: externalWide,
    onConnectionChange,
    onCommandExecuted,
}, ref) {
    const [lines, setLines] = useState<TerminalLine[]>([]);
    const [quickInput, setQuickInput] = useState('');
    const [filteredHistory, setFilteredHistory] = useState<string[]>([]);
    const [showDropdown, setShowDropdown] = useState(false);
    const [selectedIndex, setSelectedIndex] = useState(-1);
    const [wide, setWide] = useState(false);
    const [isTerminalFocused, setIsTerminalFocused] = useState(false);
    const [inAltScreen, setInAltScreen] = useState(false);
    const sessionRef = useRef<FakeShellSession | null>(null);
    const terminalInputRef = useRef<HTMLInputElement>(null);
    const quickInputRef = useRef<HTMLInputElement>(null);
    const outputRef = useRef<HTMLDivElement>(null);
    const containerRef = useRef<HTMLDivElement>(null);
    const visibleRowsRef = useRef(24);

    const isControlled = externalWide !== undefined;
    const currentWide = isControlled ? externalWide : wide;

    useEffect(() => {
        if (isControlled && externalWide !== wide) {
            setWide(externalWide);
        }
    }, [externalWide, isControlled]);

    const handleWideChange = (newWide: boolean) => {
        if (!isControlled) {
            setWide(newWide);
        }
    };

    const scrollToBottom = useCallback(() => {
        if (outputRef.current) {
            outputRef.current.scrollTop = outputRef.current.scrollHeight;
        }
    }, []);

    const calculateVisibleRows = useCallback(() => {
        if (outputRef.current) {
            const containerHeight = outputRef.current.clientHeight;
            return Math.max(Math.floor(containerHeight / LINE_HEIGHT), 5);
        }
        return 24;
    }, []);

    const syncLinesFromSession = useCallback(() => {
        if (!sessionRef.current) return;

        const outputLines = sessionRef.current.getOutputLines();
        const displayLines = outputLines.map(line => ({
            id: line.id,
            content: parseAnsiToHtml(line.content)
        }));
        setLines(displayLines);
        setInAltScreen(sessionRef.current.isInAltScreen());
        requestAnimationFrame(scrollToBottom);
    }, [scrollToBottom]);

    useEffect(() => {
        const server = getFakeShellServer();
        const session = server.createSession({ cwd, name });
        sessionRef.current = session;

        const unsubData = session.onData(() => {
            syncLinesFromSession();
        });

        const unsubClose = session.onClose(() => {
            onConnectionChange?.(false);
        });

        onConnectionChange?.(true);

        return () => {
            unsubData();
            unsubClose();
            session.close();
            sessionRef.current = null;
        };
    }, [cwd, name, onConnectionChange, syncLinesFromSession]);

    useEffect(() => {
        const outputEl = outputRef.current;
        if (!outputEl || !sessionRef.current) return;

        const updateVisibleRows = () => {
            const visibleRows = calculateVisibleRows();
            if (visibleRows !== visibleRowsRef.current && sessionRef.current) {
                visibleRowsRef.current = visibleRows;
                const cols = currentWide ? 120 : 80;
                sessionRef.current.resize(cols, visibleRows);
            }
        };

        const resizeObserver = new ResizeObserver(() => {
            updateVisibleRows();
        });

        resizeObserver.observe(outputEl);
        updateVisibleRows();

        return () => resizeObserver.disconnect();
    }, [calculateVisibleRows, currentWide]);

    useEffect(() => {
        if (sessionRef.current) {
            const cols = currentWide ? 120 : 80;
            sessionRef.current.resize(cols, visibleRowsRef.current);
        }
    }, [currentWide]);

    useEffect(() => {
        const handleResize = () => {
            if (isTerminalFocused) {
                setTimeout(() => {
                    scrollToBottom();
                    if (containerRef.current && window.visualViewport) {
                        const rect = containerRef.current.getBoundingClientRect();
                        const viewportHeight = window.visualViewport.height;
                        if (rect.bottom > viewportHeight) {
                            window.scrollTo({
                                top: window.scrollY + (rect.bottom - viewportHeight) + 20,
                                behavior: 'smooth'
                            });
                        }
                    }
                }, 100);
            }
        };

        window.addEventListener('resize', handleResize);

        if (window.visualViewport) {
            window.visualViewport.addEventListener('resize', handleResize);
        }

        return () => {
            window.removeEventListener('resize', handleResize);
            if (window.visualViewport) {
                window.visualViewport.removeEventListener('resize', handleResize);
            }
        };
    }, [isTerminalFocused, scrollToBottom]);

    const sendKey = useCallback((key: string) => {
        sessionRef.current?.send(key);
    }, []);

    useImperativeHandle(ref, () => ({
        sendKey,
        focus: () => terminalInputRef.current?.focus(),
    }), [sendKey]);

    const filterHistory = useCallback((value: string) => {
        if (!value.trim()) {
            setFilteredHistory(history.slice(0, 5));
            return history.length > 0;
        }
        const filtered = history.filter(cmd => fuzzyMatch(value, cmd));
        setFilteredHistory(filtered);
        return filtered.length > 0;
    }, [history]);

    const handleTerminalKeyDown = (e: React.KeyboardEvent<HTMLInputElement>) => {
        switch (e.key) {
            case 'Enter':
                e.preventDefault();
                sendKey('\r');
                break;
            case 'Backspace':
                e.preventDefault();
                sendKey('\b');
                break;
            case 'Delete':
                e.preventDefault();
                sendKey('\x1b[3~');
                break;
            case 'Tab':
                e.preventDefault();
                sendKey('\t');
                break;
            case 'c':
                if (e.ctrlKey) {
                    e.preventDefault();
                    sendKey('\x03');
                    break;
                }
                e.preventDefault();
                sendKey('c');
                break;
            case 'l':
                if (e.ctrlKey) {
                    e.preventDefault();
                    sendKey('\x0c');
                    break;
                }
                e.preventDefault();
                sendKey('l');
                break;
            case 'ArrowUp':
                e.preventDefault();
                sendKey('\x1b[A');
                break;
            case 'ArrowDown':
                e.preventDefault();
                sendKey('\x1b[B');
                break;
            case 'ArrowLeft':
                e.preventDefault();
                sendKey('\x1b[D');
                break;
            case 'ArrowRight':
                e.preventDefault();
                sendKey('\x1b[C');
                break;
            default:
                if (e.key.length === 1) {
                    e.preventDefault();
                    sendKey(e.key);
                }
                break;
        }
    };

    const handleTerminalFocus = () => {
        setIsTerminalFocused(true);
        setTimeout(() => {
            scrollToBottom();
            if (containerRef.current && window.visualViewport) {
                const rect = containerRef.current.getBoundingClientRect();
                const viewportHeight = window.visualViewport.height;
                if (rect.bottom > viewportHeight) {
                    window.scrollTo({
                        top: window.scrollY + (rect.bottom - viewportHeight) + 100,
                        behavior: 'smooth'
                    });
                }
            }
        }, 100);
    };

    const handleTerminalBlur = () => {
        setIsTerminalFocused(false);
    };

    const handleTerminalClick = () => {
        terminalInputRef.current?.focus();
        scrollToBottom();
    };

    const handleQuickInputChange = (value: string) => {
        setQuickInput(value);
        setSelectedIndex(-1);

        if (filterHistory(value)) {
            setShowDropdown(true);
        } else {
            setShowDropdown(false);
        }
    };

    const handleSelectCommand = (cmd: string) => {
        setQuickInput(cmd);
        setShowDropdown(false);
        setSelectedIndex(-1);
        quickInputRef.current?.focus();
    };

    const handleQuickSubmit = () => {
        if (!quickInput.trim()) return;

        const cmd = quickInput.trim();

        for (const char of cmd) {
            sendKey(char);
        }
        sendKey('\r');

        onCommandExecuted?.(cmd);
        setQuickInput('');
        setShowDropdown(false);
    };

    const handleQuickKeyDown = (e: React.KeyboardEvent) => {
        if (!showDropdown) {
            if (e.key === 'Enter') {
                e.preventDefault();
                handleQuickSubmit();
            }
            return;
        }

        switch (e.key) {
            case 'ArrowDown':
                e.preventDefault();
                setSelectedIndex(prev =>
                    prev < filteredHistory.length - 1 ? prev + 1 : prev
                );
                break;
            case 'ArrowUp':
                e.preventDefault();
                setSelectedIndex(prev => prev > 0 ? prev - 1 : -1);
                break;
            case 'Enter':
                e.preventDefault();
                if (selectedIndex >= 0 && selectedIndex < filteredHistory.length) {
                    handleSelectCommand(filteredHistory[selectedIndex]);
                } else {
                    handleQuickSubmit();
                }
                break;
            case 'Escape':
                setShowDropdown(false);
                setSelectedIndex(-1);
                break;
            case 'Tab':
                e.preventDefault();
                if (filteredHistory.length > 0) {
                    const nextIndex = selectedIndex < filteredHistory.length - 1 ? selectedIndex + 1 : 0;
                    setSelectedIndex(nextIndex);
                    setQuickInput(filteredHistory[nextIndex]);
                }
                break;
        }
    };

    const handleQuickFocus = () => {
        if (filterHistory(quickInput)) {
            setShowDropdown(true);
        }
    };

    const handleQuickBlur = () => {
        setTimeout(() => setShowDropdown(false), 150);
    };

    return (
        <div ref={containerRef} className={`custom-terminal ${className} ${currentWide ? 'custom-terminal-wide' : ''}`}>
            <div className="custom-terminal-header">
                <label className="custom-terminal-wide-toggle">
                    <input
                        type="checkbox"
                        checked={currentWide}
                        onChange={(e) => handleWideChange(e.target.checked)}
                    />
                    <span>Wide</span>
                </label>
            </div>
            <div
                className={`custom-terminal-output ${inAltScreen ? 'custom-terminal-alt-screen' : ''}`}
                ref={outputRef}
                onClick={handleTerminalClick}
                style={{ position: 'relative' }}
            >
                {lines.map((line, index) => (
                    <div
                        key={line.id}
                        className="custom-terminal-line"
                    >
                        {line.content}
                        {index === lines.length - 1 && isTerminalFocused && (
                            <span className="custom-terminal-cursor">▋</span>
                        )}
                    </div>
                ))}
                <input
                    ref={terminalInputRef}
                    type="text"
                    onKeyDown={handleTerminalKeyDown}
                    onFocus={handleTerminalFocus}
                    onBlur={handleTerminalBlur}
                    style={{
                        position: 'absolute',
                        opacity: 0,
                        pointerEvents: 'none',
                        bottom: 0,
                        left: 0,
                        width: '100%',
                        height: '20px',
                        fontSize: '16px'
                    }}
                    autoComplete="off"
                    autoCorrect="off"
                    autoCapitalize="off"
                    spellCheck={false}
                />
            </div>
            <ShortcutsBar onSendKey={sendKey} />
            <div className="custom-terminal-input-bar">
                <span className="custom-terminal-prompt">$</span>
                <div className="custom-terminal-input-wrapper">
                    <input
                        ref={quickInputRef}
                        type="text"
                        value={quickInput}
                        onChange={(e) => handleQuickInputChange(e.target.value)}
                        onKeyDown={handleQuickKeyDown}
                        onFocus={handleQuickFocus}
                        onBlur={handleQuickBlur}
                        className="custom-terminal-input"
                        placeholder="Quick input..."
                        autoComplete="off"
                        autoCorrect="off"
                        autoCapitalize="off"
                        spellCheck={false}
                    />
                    {showDropdown && filteredHistory.length > 0 && (
                        <div className="custom-terminal-dropdown">
                            {filteredHistory.map((cmd, index) => (
                                <div
                                    key={cmd}
                                    className={`custom-terminal-dropdown-item ${index === selectedIndex ? 'selected' : ''}`}
                                    onClick={() => handleSelectCommand(cmd)}
                                    onMouseDown={(e) => e.preventDefault()}
                                >
                                    <span className="custom-terminal-dropdown-prompt">$</span>
                                    <span className="custom-terminal-dropdown-text">{cmd}</span>
                                </div>
                            ))}
                        </div>
                    )}
                </div>
                <button
                    className="custom-terminal-run-btn"
                    onClick={handleQuickSubmit}
                >
                    Run
                </button>
            </div>
        </div>
    );
});
