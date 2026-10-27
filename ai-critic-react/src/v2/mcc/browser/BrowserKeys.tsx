import { useState } from 'react';

interface KeyDef {
    label: string;
    key: string;
    code: string;
    vk: number;
    wide?: boolean;
}

const KEYS: KeyDef[] = [
    { label: 'enter', key: 'Enter', code: 'Enter', vk: 13, wide: true },
    { label: 'tab', key: 'Tab', code: 'Tab', vk: 9 },
    { label: 'esc', key: 'Escape', code: 'Escape', vk: 27 },
    { label: '⌫', key: 'Backspace', code: 'Backspace', vk: 8 },
    { label: '←', key: 'ArrowLeft', code: 'ArrowLeft', vk: 37 },
    { label: '↑', key: 'ArrowUp', code: 'ArrowUp', vk: 38 },
    { label: '↓', key: 'ArrowDown', code: 'ArrowDown', vk: 40 },
    { label: '→', key: 'ArrowRight', code: 'ArrowRight', vk: 39 },
];

interface BrowserKeysProps {
    visible: boolean;
    onClose: () => void;
    onText: (text: string) => void;
    onKey: (key: string, code: string, vk: number) => void;
}

/**
 * On-screen keyboard for pages that need text.
 *
 * Text is sent as one `insertText` when you tap Send: mobile keyboards emit
 * composition and autocorrect events, and replaying those character by
 * character would double characters inside the page. Navigation keys are
 * separate so a form can be submitted without switching back to the picture.
 */
export function BrowserKeys({ visible, onClose, onText, onKey }: BrowserKeysProps) {
    const [draft, setDraft] = useState('');

    if (!visible) return null;

    const send = () => {
        if (!draft) return;
        onText(draft);
        setDraft('');
    };

    return (
        <div className="bw-keys">
            <div className="bw-keys-input">
                <input
                    value={draft}
                    onChange={(event) => setDraft(event.target.value)}
                    onKeyDown={(event) => {
                        // Enter inside the field is the Send button, not a page key.
                        if (event.key === 'Enter') {
                            event.preventDefault();
                            send();
                        }
                    }}
                    placeholder="Type text for the page…"
                    autoCapitalize="none"
                    autoCorrect="off"
                    spellCheck={false}
                />
                <button type="button" className="bw-btn" onClick={send} disabled={!draft}>
                    Send
                </button>
                <button type="button" className="bw-btn bw-btn-ghost" onClick={onClose} title="Hide keyboard">
                    ✕
                </button>
            </div>
            <div className="bw-keys-row">
                {KEYS.map((item) => (
                    <button
                        key={item.key}
                        type="button"
                        className={`bw-key${item.wide ? ' is-wide' : ''}`}
                        onClick={() => onKey(item.key, item.code, item.vk)}
                    >
                        {item.label}
                    </button>
                ))}
            </div>
        </div>
    );
}
