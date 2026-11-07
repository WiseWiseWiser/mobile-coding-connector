import { useRef } from 'react';
import type { ChangeEvent, KeyboardEvent } from 'react';

interface BrowserSearchRowProps {
    value: string;
    onChange: (value: string) => void;
    onClear: () => void;
    /** Enter jumps to the first match; the host decides what that means. */
    onEnter?: () => void;
    placeholder: string;
    disabled?: boolean;
    /** Shown on a resting field, e.g. the desktop ⌘K hint. */
    shortcut?: string;
    inputRef?: (node: HTMLInputElement | null) => void;
    /**
     * Which surface is asking. The phone row sits inside a padded wrapper and
     * fills it; the rail row is the wrapper, because the rail already has the
     * margins.
     */
    surface?: 'phone' | 'rail';
}

const SKIN = {
    phone: {
        wrap: 'bwp-sfield',
        icon: 'bwp-sicon',
        input: 'bwp-sinput',
        clear: 'bwp-sclear',
        key: 'bwp-skey',
    },
    rail: {
        wrap: 'bw-search',
        icon: 'bw-sicon',
        input: 'bw-sinput',
        clear: 'bw-sclear',
        key: 'bw-skey',
    },
} as const;

/**
 * One permanent search row. It is a row, not a view: no overlay, no extra
 * sheet height, and the field stays put while results replace the panes.
 */
export function BrowserSearchRow(props: BrowserSearchRowProps) {
    const skin = SKIN[props.surface ?? 'phone'];
    const field = useRef<HTMLInputElement>(null);
    const change = (event: ChangeEvent<HTMLInputElement>) => props.onChange(event.target.value);
    // The × unmounts itself on click, which would drop the caret on the floor;
    // clearing by hand is usually the prelude to typing again.
    const clear = () => {
        props.onClear();
        field.current?.focus();
    };
    const key = (event: KeyboardEvent<HTMLInputElement>) => {
        if (event.key === 'Enter') {
            event.preventDefault();
            props.onEnter?.();
            return;
        }
        if (event.key === 'Escape') {
            event.preventDefault();
            event.stopPropagation();
            if (props.value) props.onClear();
            else event.currentTarget.blur();
        }
    };
    return (
        <div className={skin.wrap}>
            <svg className={skin.icon} viewBox="0 0 16 16" aria-hidden="true">
                <circle cx="7" cy="7" r="4.3" />
                <path d="M10.25 10.25 13.6 13.6" />
            </svg>
            <input
                ref={(node) => {
                    field.current = node;
                    props.inputRef?.(node);
                }}
                className={skin.input}
                type="text"
                inputMode="search"
                autoComplete="off"
                autoCorrect="off"
                spellCheck={false}
                enterKeyHint="search"
                value={props.value}
                placeholder={props.placeholder}
                aria-label={props.placeholder}
                disabled={props.disabled}
                onChange={change}
                onKeyDown={key}
            />
            {props.value ? (
                <button type="button" className={skin.clear} onClick={clear} aria-label="Clear search">
                    ×
                </button>
            ) : props.shortcut ? (
                <kbd className={skin.key} aria-hidden="true">
                    {props.shortcut}
                </kbd>
            ) : null}
        </div>
    );
}
