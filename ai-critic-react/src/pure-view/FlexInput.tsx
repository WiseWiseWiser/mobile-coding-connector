import { useRef, useEffect, type CSSProperties, type ReactNode } from 'react';
import './FlexInput.css';

export interface FlexInputProps {
    value: string;
    onChange: (value: string) => void;
    type?: 'text' | 'password' | 'email';
    placeholder?: string;
    className?: string;
    inputClassName?: string;
    style?: CSSProperties;
    multiline?: boolean;
    rows?: number;
    disabled?: boolean;
    spellCheck?: boolean;
    autoFocus?: boolean;
    onKeyDown?: (e: React.KeyboardEvent) => void;
    onFocus?: (e: React.FocusEvent) => void;
    onBlur?: (e: React.FocusEvent) => void;
    children?: ReactNode;
}

export function FlexInput({
    value,
    onChange,
    type = 'text',
    placeholder,
    className = '',
    inputClassName = '',
    style,
    multiline = false,
    rows = 1,
    disabled = false,
    spellCheck = false,
    autoFocus = false,
    onKeyDown,
    onFocus,
    onBlur,
    children,
}: FlexInputProps) {
    const textareaRef = useRef<HTMLTextAreaElement>(null);

    useEffect(() => {
        if (multiline && textareaRef.current) {
            textareaRef.current.style.height = 'auto';
            textareaRef.current.style.height = textareaRef.current.scrollHeight + 'px';
        }
    }, [value, multiline]);

    const handleChange = (e: React.ChangeEvent<HTMLInputElement | HTMLTextAreaElement>) => {
        onChange(e.target.value);
    };

    const fieldClassName = `flex-input-field ${inputClassName}`.trim();

    if (multiline) {
        return (
            <div className={`flex-input-wrapper ${className}`} style={style}>
                <textarea
                    ref={textareaRef}
                    className={fieldClassName}
                    value={value}
                    onChange={handleChange}
                    placeholder={placeholder}
                    disabled={disabled}
                    spellCheck={spellCheck}
                    rows={rows}
                    autoFocus={autoFocus}
                    onKeyDown={onKeyDown as (e: React.KeyboardEvent<HTMLTextAreaElement>) => void}
                    onFocus={onFocus as (e: React.FocusEvent<HTMLTextAreaElement>) => void}
                    onBlur={onBlur as (e: React.FocusEvent<HTMLTextAreaElement>) => void}
                />
                {children}
            </div>
        );
    }

    return (
        <div className={`flex-input-wrapper ${className}`} style={style}>
            <input
                className={fieldClassName}
                type={type}
                value={value}
                onChange={handleChange}
                placeholder={placeholder}
                disabled={disabled}
                spellCheck={spellCheck}
                autoFocus={autoFocus}
                onKeyDown={onKeyDown as (e: React.KeyboardEvent<HTMLInputElement>) => void}
                onFocus={onFocus as (e: React.FocusEvent<HTMLInputElement>) => void}
                onBlur={onBlur as (e: React.FocusEvent<HTMLInputElement>) => void}
            />
            {children}
        </div>
    );
}
