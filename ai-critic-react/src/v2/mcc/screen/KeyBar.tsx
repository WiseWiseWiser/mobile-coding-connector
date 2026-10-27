import { useEffect, useRef, useState } from 'react';
import type { RefObject } from 'react';
import type { RemoteScreen } from './RemoteScreen';
import { ModeChip } from './ModeChip';
import { canSendText, draftNote, textPlaceholder } from './screenInput';
import type { ScreenMode } from './screenMode';

interface KeyBarProps {
    screen: RefObject<RemoteScreen | null>;
    platform: string;
    visible: boolean;
    mode: ScreenMode;
    /** This viewer holds the control lock and the host can take input. */
    canDrive: boolean;
    /** Bumped by every input refusal so a refused draft can come back. */
    inputErrorSeq: number;
    onToggleMode: () => void;
    onClose: () => void;
}

const SPECIAL_KEYS: Array<[code: string, label: string]> = [
    ['Escape', 'esc'],
    ['Tab', 'tab'],
    ['Backspace', '⌫'],
    ['Enter', '⏎'],
    ['Delete', 'del'],
    ['ArrowLeft', '←'],
    ['ArrowUp', '↑'],
    ['ArrowDown', '↓'],
    ['ArrowRight', '→'],
    ['Home', 'home'],
    ['End', 'end'],
    ['PageUp', 'pgup'],
    ['PageDown', 'pgdn'],
];

/**
 * How long a sent draft is held for a possible restore. A refusal comes back in
 * tens of milliseconds; past this window the text is treated as delivered, so a
 * much later error cannot resurrect something already typed on the host.
 */
const DRAFT_HOLD_MS = 3000;

/**
 * Keys the browser never delivers (⌘W, ⌘Tab, F-keys, …) plus sticky modifiers.
 * Sticky means press once to hold: the host sees a key-down with no key-up until
 * you tap it again.
 *
 * The text composer is the exception to the REMOTE/WEB split: it stays editable
 * in WEB so a note can be written while browsing, but it only sends in REMOTE.
 * A draft that could not be sent is never thrown away — not by a mode flip, not
 * by hiding the bar, and not by the server refusing the frame.
 */
export function KeyBar({
    screen, platform, visible, mode, canDrive, inputErrorSeq, onToggleMode, onClose,
}: KeyBarProps) {
    const [held, setHeld] = useState<Record<string, boolean>>({});
    const [text, setText] = useState('');
    // The last draft handed to the socket, kept until it is either refused
    // (restored below) or superseded by an edit or another send.
    const pending = useRef<string | null>(null);
    const pendingTimer = useRef<number | null>(null);

    const isMac = platform === 'darwin';
    // WEB mode owns every keystroke, so the host keys would be dead weight —
    // dim them instead of letting people press buttons that cannot work.
    const live = mode === 'remote';
    const sendable = canSendText(mode, canDrive, text);
    const note = draftNote(mode, canDrive, text);
    const modifiers: Array<[code: string, label: string]> = [
        ['ControlLeft', isMac ? '⌃' : 'ctrl'],
        ['AltLeft', isMac ? '⌥' : 'alt'],
        ['MetaLeft', isMac ? '⌘' : 'meta'],
        ['ShiftLeft', '⇧'],
    ];

    const forgetPending = () => {
        pending.current = null;
        if (pendingTimer.current !== null) {
            window.clearTimeout(pendingTimer.current);
            pendingTimer.current = null;
        }
    };

    useEffect(() => () => {
        if (pendingTimer.current !== null) window.clearTimeout(pendingTimer.current);
    }, []);

    // A refusal is the only proof that a frame was dropped (the protocol has no
    // ack), so it is also the only moment a draft may reappear.
    useEffect(() => {
        if (inputErrorSeq === 0) return;
        const draft = pending.current;
        if (draft === null) return;
        pending.current = null;
        if (pendingTimer.current !== null) {
            window.clearTimeout(pendingTimer.current);
            pendingTimer.current = null;
        }
        setText(draft);
    }, [inputErrorSeq]);

    const toggleModifier = (code: string) => {
        const remote = screen.current;
        if (!remote) return;
        const next = !held[code];
        remote.key(code, next);
        setHeld((prev) => ({ ...prev, [code]: next }));
    };

    const tapKey = (code: string) => {
        const remote = screen.current;
        if (!remote) return;
        remote.key(code, true);
        remote.key(code, false);
    };

    const releaseHeld = () => {
        const remote = screen.current;
        if (!remote) return;
        for (const code of Object.keys(held)) {
            if (held[code]) remote.key(code, false);
        }
        setHeld({});
    };

    const send = () => {
        if (!sendable) return;
        const remote = screen.current;
        // Nothing was written, so the draft stays where the user can see it.
        if (!remote) return;
        remote.text(text);
        pending.current = text;
        if (pendingTimer.current !== null) window.clearTimeout(pendingTimer.current);
        pendingTimer.current = window.setTimeout(() => {
            pending.current = null;
            pendingTimer.current = null;
        }, DRAFT_HOLD_MS);
        setText('');
    };

    if (!visible) return null;

    return (
        <div className={`screen-keys${live ? '' : ' screen-keys-web'}`}>
            <div className="screen-keys-row">
                <ModeChip mode={mode} onToggle={onToggleMode} className="screen-key screen-key-wide screen-key-mode" />
                {modifiers.map(([code, label]) => (
                    <button
                        key={code}
                        className={`screen-key ${held[code] ? 'is-held' : ''}`}
                        disabled={!live}
                        onClick={() => toggleModifier(code)}
                    >
                        {label}
                    </button>
                ))}
                <button className="screen-key screen-key-wide" disabled={!live} onClick={releaseHeld}>
                    release keys
                </button>
                <button className="screen-key screen-key-wide" onClick={onClose}>
                    hide
                </button>
            </div>
            <div className="screen-keys-row screen-keys-scroll">
                {SPECIAL_KEYS.map(([code, label]) => (
                    <button key={code} className="screen-key" disabled={!live} onClick={() => tapKey(code)}>
                        {label}
                    </button>
                ))}
            </div>
            <div className="screen-keys-row">
                <input
                    className="screen-text-input"
                    value={text}
                    placeholder={textPlaceholder(mode, canDrive, isMac)}
                    onChange={(event) => {
                        setText(event.target.value);
                        // An edit supersedes whatever we were holding for a retry.
                        forgetPending();
                    }}
                    onKeyDown={(event) => {
                        if (event.key === 'Escape') {
                            // Browsing: drop focus, keep the draft. Driving: the
                            // old "clear and get out of the way" still holds.
                            if (!live) {
                                event.currentTarget.blur();
                                return;
                            }
                            setText('');
                            forgetPending();
                            onClose();
                            return;
                        }
                        if (event.key !== 'Enter') return;
                        event.preventDefault();
                        // WEB or view-only: the draft is not sent and not lost.
                        if (!sendable) return;
                        send();
                    }}
                />
                <button className="screen-key" disabled={!sendable} onClick={send}>
                    send
                </button>
            </div>
            {note && <div className="screen-keys-note">{note}</div>}
        </div>
    );
}
