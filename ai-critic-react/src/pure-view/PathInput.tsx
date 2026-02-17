import { useRef, useEffect } from 'react';
import './PathInput.css';

export interface PathInputProps {
    value: string;
    onChange: (value: string) => void;
    label?: string;
}

export function PathInput({ value, onChange, label = 'Project Directory' }: PathInputProps) {
    const textareaRef = useRef<HTMLTextAreaElement>(null);

    useEffect(() => {
        if (textareaRef.current) {
            textareaRef.current.style.height = 'auto';
            textareaRef.current.style.height = textareaRef.current.scrollHeight + 'px';
        }
    }, [value]);

    return (
        <div className="path-input-container">
            {label && <h3 className="path-input-label">{label}</h3>}
            <textarea
                ref={textareaRef}
                className="path-input-field"
                value={value}
                onChange={(e) => onChange(e.target.value)}
                spellCheck={false}
                rows={1}
            />
        </div>
    );
}
