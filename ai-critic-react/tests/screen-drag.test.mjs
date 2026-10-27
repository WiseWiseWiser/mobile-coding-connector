import assert from 'node:assert/strict';
import { test } from 'node:test';
import { DRAG_HOLD_MS, DragLatch, canStartDragHold } from '../src/v2/mcc/screen/screenDrag.ts';

/** Records what the latch did to the host button. */
function host() {
    const calls = [];
    return {
        calls,
        press: () => calls.push('down'),
        release: () => calls.push('up'),
    };
}

const settle = (ms) => new Promise((resolve) => setTimeout(resolve, ms));

test('a single finger in remote mode with control may grab', () => {
    assert.equal(canStartDragHold('remote', true, 1), true);
});

test('browsing never grabs — the picture pans instead', () => {
    assert.equal(canStartDragHold('web', true, 1), false);
    assert.equal(canStartDragHold('web', false, 1), false);
});

test('without control there is no host cursor to grab', () => {
    assert.equal(canStartDragHold('remote', false, 1), false);
});

test('two fingers scroll, they never grab', () => {
    assert.equal(canStartDragHold('remote', true, 2), false);
    assert.equal(canStartDragHold('remote', true, 0), false);
});

test('a quick tap never grabs', async () => {
    const h = host();
    const grabbed = [];
    const latch = new DragLatch(h, () => true, (held) => grabbed.push(held), 20);
    latch.arm();
    await settle(5);
    latch.drop(true);
    await settle(40);
    assert.deepEqual(h.calls, []);
    assert.deepEqual(grabbed, []);
    assert.equal(latch.active, false);
});

test('resting a finger grabs, and the lift drops it', async () => {
    const h = host();
    const grabbed = [];
    const latch = new DragLatch(h, () => true, (held) => grabbed.push(held), 20);
    latch.arm();
    await settle(50);
    assert.deepEqual(h.calls, ['down']);
    assert.equal(latch.active, true);
    latch.drop(true);
    assert.deepEqual(h.calls, ['down', 'up']);
    assert.deepEqual(grabbed, [true, false]);
    assert.equal(latch.active, false);
});

test('a finger that wanders before the hold never grabs', async () => {
    const h = host();
    let moved = false;
    const latch = new DragLatch(h, () => !moved, () => {}, 20);
    latch.arm();
    moved = true;
    await settle(50);
    assert.deepEqual(h.calls, []);
    assert.equal(latch.active, false);
});

test('a second finger cancels a pending grab but drops a live one', async () => {
    const h = host();
    const pending = new DragLatch(h, () => true, () => {}, 20);
    pending.arm();
    pending.cancel();
    await settle(50);
    assert.deepEqual(h.calls, []);

    const live = new DragLatch(h, () => true, () => {}, 20);
    live.arm();
    await settle(50);
    live.drop(true);
    assert.deepEqual(h.calls, ['down', 'up']);
});

test('losing control drops locally without lifting the host button', async () => {
    const h = host();
    const grabbed = [];
    const latch = new DragLatch(h, () => true, (held) => grabbed.push(held), 20);
    latch.arm();
    await settle(50);
    latch.drop(false);
    assert.deepEqual(h.calls, ['down']);
    assert.deepEqual(grabbed, [true, false]);
});

test('dropping twice is a no-op', async () => {
    const h = host();
    const latch = new DragLatch(h, () => true, () => {}, 20);
    latch.arm();
    await settle(50);
    latch.drop(true);
    latch.drop(true);
    assert.deepEqual(h.calls, ['down', 'up']);
});

test('the hold length is a parameter, not a constant', async () => {
    assert.equal(DRAG_HOLD_MS, 350);
    const h = host();
    const latch = new DragLatch(h, () => true, () => {}, 10);
    latch.arm();
    await settle(40);
    assert.deepEqual(h.calls, ['down']);
});
