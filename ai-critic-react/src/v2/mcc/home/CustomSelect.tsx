import { useState, useRef, useEffect } from 'react';
import './CustomSelect.css';

interface CustomSelectOption {
    value: string;
    label: string;
    sublabel?: string;
}

interface CustomSelectProps {
    value: string;
    onChange: (value: string) => void;
    options: CustomSelectOption[];
    placeholder?: string;
}

export function CustomSelect({ value, onChange, options, placeholder = 'Select...' }: CustomSelectProps) {
    const [open, setOpen] = useState(false);
    const containerRef = useRef<HTMLDivElement>(null);

    // Close on outside click
    useEffect(() => {
        if (!open) return;
        const handleClick = (e: MouseEvent) => {
            if (containerRef.current && !containerRef.current.contains(e.target as Node)) {
                setOpen(false);
            }
        };
        document.addEventListener('mousedown', handleClick);
        return () => document.removeEventListener('mousedown', handleClick);
    }, [open]);

    const selectedOption = options.find(o => o.value === value);

    return (
        <div className="custom-select" ref={containerRef}>
            <button
                className={`custom-select-trigger${open ? ' custom-select-trigger--open' : ''}`}
                onClick={() => setOpen(!open)}
                type="button"
            >
                <span className="custom-select-trigger-text">
                    {selectedOption ? selectedOption.label : placeholder}
                </span>
                <span className="custom-select-arrow">{open ? '▲' : '▼'}</span>
            </button>
            {open && (
                <div className="custom-select-dropdown">
                    {options.map(opt => (
                        <button
                            key={opt.value}
                            className={`custom-select-option${opt.value === value ? ' custom-select-option--selected' : ''}`}
                            onClick={() => {
                                onChange(opt.value);
                                setOpen(false);
                            }}
                            type="button"
                        >
                            <span className="custom-select-option-label">{opt.label}</span>
                            {opt.sublabel && (
                                <span className="custom-select-option-sublabel">{opt.sublabel}</span>
                            )}
                            {opt.value === value && (
                                <span className="custom-select-option-check">✓</span>
                            )}
                        </button>
                    ))}
                </div>
            )}
        </div>
    );
}
