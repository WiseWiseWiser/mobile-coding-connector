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
    maxLines?: number;
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

const MAX_LINES = 256;

let lineIdCounter = 0;

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

function applyBackspace(line: string, backspaceSeq: string): string {
    let result = line;
    let i = 0;
    while (i < backspaceSeq.length) {
        const char = backspaceSeq[i];
        if (char === '\b' || char === '\x7f') {
            // Check for \b \b pattern (backspace + space + backspace)
            // This is a common terminal sequence for deleting a character
            if (i + 2 < backspaceSeq.length && backspaceSeq[i + 1] === ' ' && backspaceSeq[i + 2] === '\b') {
                // Treat as single backspace
                result = result.slice(0, -1);
                i += 3;
            } else {
                result = result.slice(0, -1);
                i++;
            }
        } else {
            i++;
        }
    }
    return result;
}

export const CustomTerminal = forwardRef<CustomTerminalHandle, CustomTerminalProps>(function CustomTerminal({
    className = '',
    cwd = '/home/user',
    name = 'mock-shell',
    history,
    wide: externalWide,
    maxLines = MAX_LINES,
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
    const sessionRef = useRef<FakeShellSession | null>(null);
    const terminalInputRef = useRef<HTMLInputElement>(null);
    const quickInputRef = useRef<HTMLInputElement>(null);
    const outputRef = useRef<HTMLDivElement>(null);
    const containerRef = useRef<HTMLDivElement>(null);
    // Use a ref to store all lines including the current incomplete one
    const allLinesRef = useRef<{ id: number; content: string }[]>([]);
    const currentLineContentRef = useRef('');

    const isControlled = externalWide !== undefined;
    const currentWide = isControlled ? externalWide : wide;

    useEffect(() => {
        if (isControlled && externalWide !== wide) {
            setWide(externalWide);
        }
    }, [externalWide, isControlled]);

    useEffect(() => {
        if (sessionRef.current) {
            const newCols = currentWide ? 120 : 80;
            sessionRef.current.resize(newCols, 24);
        }
    }, [currentWide]);

    const handleWideChange = (newWide: boolean) => {
        if (!isControlled) {
            setWide(newWide);
        }
    };

    // Scroll to bottom
    const scrollToBottom = useCallback(() => {
        if (outputRef.current) {
            outputRef.current.scrollTop = outputRef.current.scrollHeight;
        }
    }, []);

    // Sync lines ref to React state
    const syncLines = useCallback(() => {
        const displayLines = allLinesRef.current.map(line => ({
            id: line.id,
            content: parseAnsiToHtml(line.content)
        }));
        setLines(displayLines);
        requestAnimationFrame(scrollToBottom);
    }, [scrollToBottom]);

    useEffect(() => {
        const server = getFakeShellServer();
        const session = server.createSession({ cwd, name });
        sessionRef.current = session;

        const unsubData = session.onData((data) => {
            // Process backspace sequences first
            let processedData = data;
            
            // Handle backspace sequences like '\b \b'
            if (processedData.includes('\b')) {
                // Apply backspace processing to current line
                currentLineContentRef.current = applyBackspace(currentLineContentRef.current, processedData);
                // Update the last line in display
                if (allLinesRef.current.length > 0) {
                    allLinesRef.current[allLinesRef.current.length - 1].content = currentLineContentRef.current;
                }
                syncLines();
                return;
            }
            
            // Check if data contains newline
            if (processedData.includes('\r\n')) {
                // Has newline - split and process
                const parts = processedData.split('\r\n');
                
                // First part combines with current line content and becomes complete
                const firstPart = currentLineContentRef.current + parts[0];
                if (allLinesRef.current.length > 0 && currentLineContentRef.current.length > 0) {
                    // Update the last line
                    allLinesRef.current[allLinesRef.current.length - 1].content = firstPart;
                } else if (firstPart.length > 0) {
                    // Add new line
                    allLinesRef.current.push({ id: ++lineIdCounter, content: firstPart });
                }
                
                // Reset current line
                currentLineContentRef.current = '';
                
                // Middle parts are complete lines
                for (let i = 1; i < parts.length - 1; i++) {
                    if (parts[i].length > 0) {
                        allLinesRef.current.push({ id: ++lineIdCounter, content: parts[i] });
                    }
                }
                
                // Last part becomes the new current line
                const lastPart = parts[parts.length - 1];
                if (lastPart.length > 0) {
                    currentLineContentRef.current = lastPart;
                    allLinesRef.current.push({ id: ++lineIdCounter, content: lastPart });
                }
            } else if (processedData.length > 0) {
                // No newline - append to current line
                const wasEmpty = currentLineContentRef.current.length === 0;
                currentLineContentRef.current += processedData;
                
                if (wasEmpty) {
                    // Starting a new line - add it
                    allLinesRef.current.push({ id: ++lineIdCounter, content: currentLineContentRef.current });
                } else if (allLinesRef.current.length > 0) {
                    // Update last line
                    allLinesRef.current[allLinesRef.current.length - 1].content = currentLineContentRef.current;
                } else {
                    // First line
                    allLinesRef.current.push({ id: ++lineIdCounter, content: currentLineContentRef.current });
                }
            }
            
            // Enforce max lines
            if (allLinesRef.current.length > maxLines) {
                allLinesRef.current = allLinesRef.current.slice(-maxLines);
            }
            
            // Sync to React state
            syncLines();
        });

        const unsubClose = session.onClose(() => {
            onConnectionChange?.(false);
        });

        const newCols = currentWide ? 120 : 80;
        session.resize(newCols, 24);

        onConnectionChange?.(true);

        return () => {
            unsubData();
            unsubClose();
            session.close();
            sessionRef.current = null;
        };
    }, [cwd, name, maxLines, onConnectionChange, syncLines]);

    useEffect(() => {
        if (sessionRef.current) {
            const newCols = currentWide ? 120 : 80;
            sessionRef.current.resize(newCols, 24);
        }
    }, [currentWide]);

    // Handle iOS Safari keyboard visibility
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

    // Handle keystrokes from hidden input
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

    // Handle quick input (bottom input box)
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
        
        // Send command through shell
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
                className="custom-terminal-output" 
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
                {/* Hidden input for capturing keystrokes */}
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
