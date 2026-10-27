import type { ScreenStatus } from '../../../api/screen';
import type { ScreenMode } from './screenMode';

/**
 * The error codes the server sends when an input frame arrives that it will not
 * dispatch. Everything else on the error channel (a stall notice, a release
 * broadcast carrying a reason) is news about the connection, not about the
 * frame we just wrote — so only these codes may bring a draft back.
 */
const INPUT_ERROR_CODES: readonly string[] = ['not_controller', 'host_locked', 'input_failed'];

export function isInputErrorCode(code: string | undefined): boolean {
    return code !== undefined && INPUT_ERROR_CODES.includes(code);
}

/**
 * Whether this page may inject anything at all, judged from the status frame
 * alone. These are the refusals the server applies before it even decodes the
 * frame, so a key bar that checks them first never provokes an error it cannot
 * act on.
 */
export function controlPossible(status: ScreenStatus | null): boolean {
    if (!status) return false;
    return status.control.enabled && status.input.available && !status.input.locked;
}

/** True when this viewer holds the lock and injection is possible right now. */
export function canDriveHost(status: ScreenStatus | null, holdingControl: boolean): boolean {
    return holdingControl && controlPossible(status);
}

/**
 * The send gate. Text is composed in both modes, but it only leaves the device
 * in REMOTE while this viewer is driving: in WEB every gesture belongs to the
 * browser, and without the lock the server would refuse the frame anyway.
 */
export function canSendText(mode: ScreenMode, canDrive: boolean, text: string): boolean {
    return mode === 'remote' && canDrive && text !== '';
}

/**
 * What to do about a draft that cannot leave the device right now. Taking
 * control is listed before switching mode because it grants the lock and flips
 * the mode in one step; switching mode alone would still not send anything.
 */
function sendHint(mode: ScreenMode, canDrive: boolean): string | null {
    if (!canDrive) return 'take control to send';
    if (mode === 'web') return 'switch to REMOTE to send';
    return null;
}

/**
 * Placeholder for the composer. The host-text hint is only honest when a send
 * would actually land, so the other states say what is missing instead.
 */
export function textPlaceholder(mode: ScreenMode, canDrive: boolean, isMac: boolean): string {
    if (canDrive && mode === 'remote') {
        return isMac ? 'Type text for the host (IME, emoji, 中文…)' : 'Type text for the host (IME, emoji…)';
    }
    const hint = sendHint(mode, canDrive);
    return hint === null ? '' : `Composing only — ${hint}`;
}

/**
 * One line under the composer explaining why a typed draft is still sitting
 * here. Null when there is nothing to explain (no draft, or the draft can go).
 */
export function draftNote(mode: ScreenMode, canDrive: boolean, text: string): string | null {
    if (text === '' || canSendText(mode, canDrive, text)) return null;
    const hint = sendHint(mode, canDrive);
    return hint === null ? null : `draft — ${hint}`;
}
