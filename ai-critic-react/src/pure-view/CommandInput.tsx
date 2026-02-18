import { useState, useRef, useCallback } from 'react';
import './CommandInput.css';

export type DropdownPosition = 'top' | 'bottom';

export interface CommandInputProps {
    value: string;
    onChange: (value: string) => void;
    onSubmit: (command: string) => void;
    history?: string[];
    placeholder?: string;
    className?: string;
    dropdownPosition?: DropdownPosition;
}

export function CommandInput({
    value,
    onChange,
    onSubmit,
    history = [],
    placeholder = 'Type a command...',
    className = '',
    dropdownPosition = 'bottom',
}: CommandInputProps) {
    const [showDropdown, setShowDropdown] = useState(false);
    const [filteredHistory, setFilteredHistory] = useState<string[]>([]);
    const [selectedIndex, setSelectedIndex] = useState(-1);
    const inputRef = useRef<HTMLInputElement>(null);
    const dropdownRef = useRef<HTMLDivElement>(null);

    const fuzzyMatch = useCallback((query: string, candidate: string): boolean => {
        const q = query.toLowerCase();
        const c = candidate.toLowerCase();
        let qi = 0;
        for (let i = 0; i < c.length && qi < q.length; i++) {
            if (c[i] === q[qi]) qi++;
        }
        return qi === q.length;
    }, []);

    const filterHistory = useCallback((inputValue: string) => {
        if (!inputValue.trim()) {
            setFilteredHistory(history.slice(0, 5));
            return history.length > 0;
        }
        const filtered = history.filter(cmd => fuzzyMatch(inputValue, cmd));
        setFilteredHistory(filtered);
        return filtered.length > 0;
    }, [history, fuzzyMatch]);

    const handleInputChange = (newValue: string) => {
        onChange(newValue);
        setSelectedIndex(-1);
        
        if (filterHistory(newValue)) {
            setShowDropdown(true);
        } else {
            setShowDropdown(false);
        }
    };

    const handleSelectCommand = (cmd: string) => {
        onChange(cmd);
        setShowDropdown(false);
        setSelectedIndex(-1);
        inputRef.current?.focus();
    };

    const handleKeyDown = (e: React.KeyboardEvent) => {
        if (!showDropdown) {
            if (e.key === 'Enter') {
                e.preventDefault();
                if (value.trim()) {
                    onSubmit(value.trim());
                }
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
                } else if (value.trim()) {
                    onSubmit(value.trim());
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
                    onChange(filteredHistory[nextIndex]);
                }
                break;
        }
    };

    const handleFocus = () => {
        if (filterHistory(value)) {
            setShowDropdown(true);
        }
    };

    const handleBlur = () => {
        setTimeout(() => setShowDropdown(false), 150);
    };

    return (
        <div className={`command-input-container ${className}`}>
            <div className="command-input-wrapper" ref={dropdownRef}>
                <span className="command-prompt">$</span>
                <div className="command-input-inner">
                    <input
                        ref={inputRef}
                        type="text"
                        value={value}
                        onChange={(e) => handleInputChange(e.target.value)}
                        onKeyDown={handleKeyDown}
                        onFocus={handleFocus}
                        onBlur={handleBlur}
                        className="command-input-field"
                        placeholder={placeholder}
                        autoComplete="off"
                        autoCorrect="off"
                        autoCapitalize="off"
                        spellCheck={false}
                    />
                    {showDropdown && filteredHistory.length > 0 && (
                        <div className={`command-dropdown command-dropdown-${dropdownPosition}`}>
                            {filteredHistory.map((cmd, index) => (
                                <div
                                    key={cmd}
                                    className={`command-dropdown-item ${index === selectedIndex ? 'selected' : ''}`}
                                    onClick={() => handleSelectCommand(cmd)}
                                    onMouseDown={(e) => e.preventDefault()}
                                >
                                    <span className="command-dropdown-prompt">$</span>
                                    <span className="command-dropdown-text">{cmd}</span>
                                </div>
                            ))}
                        </div>
                    )}
                </div>
            </div>
        </div>
    );
}
