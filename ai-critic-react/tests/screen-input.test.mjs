import assert from 'node:assert/strict';
import { test } from 'node:test';
import {
    canDriveHost,
    canSendText,
    controlPossible,
    draftNote,
    isInputErrorCode,
    textPlaceholder,
} from '../src/v2/mcc/screen/screenInput.ts';

/** Minimal status objects: only the fields the predicate reads. */
function status({ enabled = true, available = true, locked = false } = {}) {
    return {
        state: 'streaming',
        platform: 'darwin',
        displays: null,
        capture: { tool: 'screencap', width: 1280, height: 826, fps: 12, idle_fps: 3, tile_size: 128 },
        control: { enabled, holder: '', since: 0 },
        input: { available, reason: available ? undefined : 'no accessibility', can_prompt: true, backend: 'cgevent', locked },
        viewers: 1,
    };
}

test('only the three refusal codes may restore a draft', () => {
    for (const code of ['not_controller', 'host_locked', 'input_failed']) {
        assert.equal(isInputErrorCode(code), true, code);
    }
    // Release broadcasts and connection news are not proofs of a dropped frame.
    for (const code of ['released', 'error', 'NOT_CONTROLLER', '']) {
        assert.equal(isInputErrorCode(code), false, code);
    }
    assert.equal(isInputErrorCode(undefined), false);
});

test('controlPossible mirrors the three server-side refusals', () => {
    assert.equal(controlPossible(null), false, 'no status yet');
    assert.equal(controlPossible(status()), true, 'enabled, available, unlocked');
    assert.equal(controlPossible(status({ enabled: false })), false, 'control disabled by config');
    assert.equal(controlPossible(status({ available: false })), false, 'no input backend');
    assert.equal(controlPossible(status({ locked: false })), true, 'unlocked host');
    assert.equal(controlPossible(status({ locked: true })), false, 'locked host');
});

test('canDriveHost needs the lock and a usable host', () => {
    assert.equal(canDriveHost(status(), true), true);
    assert.equal(canDriveHost(status(), false), false, 'viewing without the lock');
    assert.equal(canDriveHost(status({ locked: true }), true), false);
    assert.equal(canDriveHost(status({ available: false }), true), false);
    assert.equal(canDriveHost(null, true), false, 'status not loaded');
});

test('text is only sent from REMOTE while driving', () => {
    assert.equal(canSendText('remote', true, 'hi'), true);
    assert.equal(canSendText('web', true, 'hi'), false, 'WEB gestures belong to the browser');
    assert.equal(canSendText('remote', false, 'hi'), false, 'no lock, nothing to inject into');
    assert.equal(canSendText('web', false, 'hi'), false);
    assert.equal(canSendText('remote', true, ''), false, 'empty send is a no-op');
    assert.equal(canSendText('remote', true, ' '), true, 'a space is a real keystroke');
});

test('placeholder names whatever is missing', () => {
    assert.match(textPlaceholder('remote', true, true), /^Type text for the host/);
    assert.match(textPlaceholder('remote', true, false), /^Type text for the host/);
    assert.equal(textPlaceholder('web', true, true), 'Composing only — switch to REMOTE to send');
    assert.equal(textPlaceholder('remote', false, true), 'Composing only — take control to send');
    assert.equal(textPlaceholder('web', false, true), 'Composing only — take control to send');
});

test('the draft note explains a stuck draft and nothing else', () => {
    assert.equal(draftNote('web', false, ''), null, 'no draft, no note');
    assert.equal(draftNote('remote', true, 'hi'), null, 'sendable');
    assert.equal(draftNote('web', true, 'hi'), 'draft — switch to REMOTE to send');
    assert.equal(draftNote('remote', false, 'hi'), 'draft — take control to send');
    // Taking control also flips the mode, so it is the right advice in both.
    assert.equal(draftNote('web', false, 'hi'), 'draft — take control to send');
});

test('the note is shown exactly when a send would be dropped', () => {
    for (const mode of ['remote', 'web']) {
        for (const canDrive of [true, false]) {
            for (const text of ['', 'x']) {
                const stuck = text !== '' && !canSendText(mode, canDrive, text);
                assert.equal(draftNote(mode, canDrive, text) !== null, stuck, `${mode}/${canDrive}/"${text}"`);
            }
        }
    }
});
