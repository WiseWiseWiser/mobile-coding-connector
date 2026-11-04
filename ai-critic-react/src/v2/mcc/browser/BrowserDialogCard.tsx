import { useEffect, useRef, useState } from 'react';
import type { JsDialog } from './browserWire';

interface BrowserDialogCardProps {
    dialog: JsDialog;
    /** accept=true presses OK; false presses Cancel. text answers a prompt. */
    onAnswer: (accept: boolean, text?: string) => void;
}

/**
 * A JS dialog (alert/confirm/prompt/beforeunload) opened by the driven page.
 * The page is suspended until the dialog is answered, so the mirror shows a
 * card instead of waiting on frames: the last frame stays visible underneath.
 */
export function BrowserDialogCard({ dialog, onAnswer }: BrowserDialogCardProps) {
    const [text, setText] = useState(dialog.default_prompt || '');
    const inputRef = useRef<HTMLInputElement | null>(null);
    useEffect(() => {
        setText(dialog.default_prompt || '');
    }, [dialog.default_prompt, dialog.message]);
    useEffect(() => {
        if (dialog.type === 'prompt') inputRef.current?.focus();
    }, [dialog.type]);

    let host = '';
    try {
        host = dialog.url ? new URL(dialog.url).hostname : '';
    } catch {
        host = '';
    }
    const answer = (accept: boolean) => onAnswer(accept, dialog.type === 'prompt' ? text : undefined);
    const label = dialog.type === 'beforeunload' ? 'Leave site?' : host ? `${host} says:` : 'This page says:';

    return (
        <div className="bw-dialog" role="alertdialog" aria-label="Page dialog">
            <div className="bw-dialog-card">
                <div className="bw-dialog-host">{label}</div>
                {dialog.message && <p className="bw-dialog-msg">{dialog.message}</p>}
                {dialog.type === 'prompt' && (
                    <input
                        ref={inputRef}
                        className="bw-dialog-input"
                        value={text}
                        onChange={(event) => setText(event.target.value)}
                        onKeyDown={(event) => {
                            if (event.key === 'Enter') answer(true);
                        }}
                        aria-label="Prompt input"
                    />
                )}
                <div className="bw-dialog-btns">
                    {dialog.type !== 'alert' && (
                        <button type="button" className="bw-dialog-btn" onClick={() => answer(false)}>
                            Cancel
                        </button>
                    )}
                    <button type="button" className="bw-dialog-btn is-primary" onClick={() => answer(true)}>
                        OK
                    </button>
                </div>
            </div>
        </div>
    );
}
