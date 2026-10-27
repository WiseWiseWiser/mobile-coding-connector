import assert from 'node:assert/strict';
import { test } from 'node:test';
import { centreView, panBy, scaleOf, zoomAt } from '../src/v2/mcc/screen/viewport.ts';
import {
    contentTabs,
    emptyGeometry,
    geometryOf,
    hasGeometry,
    localPoint,
    normalizeAt,
    resetView,
    screenAt,
    settleView,
    transformOf,
    transformStyle,
} from '../src/v2/mcc/browser/browserView.ts';

// A phone-sized capture in a desktop-sized stage, the shape /browser actually
// sees when an iPhone preset is driven from a laptop.
const stage = { left: 100, top: 40, width: 900, height: 760 };
const image = { width: 500, height: 844 };
const g = geometryOf(stage, image.width, image.height);

const near = (actual, expected, tolerance = 1e-6) =>
    assert.ok(Math.abs(actual - expected) <= tolerance, `${actual} ≉ ${expected}`);

test('geometry needs a stage and a decoded frame', () => {
    assert.equal(hasGeometry(g), true);
    assert.equal(hasGeometry(emptyGeometry()), false);
    assert.equal(hasGeometry({ ...g, imgW: 0 }), false);
    // 500×844 in a 900×760 stage is height-limited.
    near(Math.min(g.stageW / g.imgW, g.stageH / g.imgH), 760 / 844);
});

test('1× view is the untouched fitted frame', () => {
    const t = transformOf(g, resetView(g));
    near(t.tx, 0);
    near(t.ty, 0);
    near(t.scale, 1);
    assert.equal(transformStyle(t), 'none');
});

test('zoom about the centre keeps the picture centred', () => {
    const t = transformOf(g, { zoom: 2.5, cx: g.imgW / 2, cy: g.imgH / 2 });
    near(t.tx, 0);
    near(t.ty, 0);
    near(t.scale, 2.5);
    assert.match(transformStyle(t), /^translate\(0\.0px, 0\.0px\) scale\(2\.500\)$/);
});

test('panning right moves the frame left by the same distance on screen', () => {
    const k = (760 / 844) * 3;
    const view = settleView(g, { zoom: 3, cx: g.imgW / 2 + 50, cy: g.imgH / 2 });
    const t = transformOf(g, view);
    near(t.tx, -50 * k);
    near(t.ty, 0);
});

test('localPoint removes the stage origin', () => {
    assert.deepEqual(localPoint(stage, 100, 40), { x: 0, y: 0 });
    assert.deepEqual(localPoint(stage, 550, 420), { x: 450, y: 380 });
});

test('normalizeAt is the inverse of screenAt', () => {
    const views = [
        resetView(g),
        { zoom: 2.5, cx: g.imgW / 2, cy: g.imgH / 2 },
        settleView(g, { zoom: 1.8, cx: g.imgW * 0.72, cy: g.imgH * 0.3 }),
    ];
    const points = [
        { x: 0, y: 0 },
        { x: 0.5, y: 0.5 },
        { x: 1, y: 1 },
        { x: 0.13, y: 0.87 },
    ];
    for (const view of views) {
        for (const point of points) {
            const screen = screenAt(g, view, point);
            const back = normalizeAt(g, view, screen);
            near(back.x, point.x, 1e-6);
            near(back.y, point.y, 1e-6);
        }
    }
});

test('normalizeAt follows a panned view, not the stage', () => {
    const view = settleView(g, { zoom: 4, cx: g.imgW / 2, cy: g.imgH / 2 });
    const centre = { x: g.stageW / 2, y: g.stageH / 2 };
    const dragged = panBy(g, view, -60, -80);
    // Dragging the picture left/up slides the view right/down: the same stage
    // point now sits on a different part of the page.
    const before = normalizeAt(g, view, centre);
    const after = normalizeAt(g, dragged, centre);
    assert.ok(after.x > before.x);
    assert.ok(after.y > before.y);
    near(after.x - before.x, 60 / (scaleOf(g, dragged) * g.imgW), 1e-6);
    near(after.y - before.y, 80 / (scaleOf(g, dragged) * g.imgH), 1e-6);
});

test('zooming on an anchor keeps that page point under the finger', () => {
    const anchor = { x: 500, y: 460 };
    const before = normalizeAt(g, resetView(g), anchor);
    const imagePx = { x: before.x * g.imgW, y: before.y * g.imgH };
    const zoomed = zoomAt(g, resetView(g), 3.2, imagePx);
    const after = normalizeAt(g, zoomed, anchor);
    near(after.x, before.x, 1e-3);
    near(after.y, before.y, 1e-3);
    near(zoomed.zoom, 3.2);
});

test('letterbox taps clamp to the nearest page edge', () => {
    // 900×760 stage, image fitted to 450×760: the left band is page left.
    const left = normalizeAt(g, resetView(g), { x: 0, y: 380 });
    assert.deepEqual(left, { x: 0, y: 380 / 760 });
    const right = normalizeAt(g, resetView(g), { x: 900, y: 380 });
    assert.equal(right.x, 1);
    const top = normalizeAt(g, resetView(g), { x: 450, y: 0 });
    assert.equal(top.y, 0);
});

test('normalizeAt without geometry lands in the middle', () => {
    assert.deepEqual(normalizeAt(emptyGeometry(), { zoom: 1, cx: 0, cy: 0 }, { x: 10, y: 20 }), { x: 0.5, y: 0.5 });
});

test('resetView is 1× on the picture centre even for odd sizes', () => {
    assert.deepEqual(resetView(g), centreView(g));
    assert.deepEqual(resetView(emptyGeometry()), { zoom: 1, cx: 0, cy: 0 });
});

test('contentTabs drops the /go session page and keeps the rest in order', () => {
    const tabs = [
        { tab_id: 1, index: 0, url: 'http://127.0.0.1:43761/go?session=ai-critic', title: 'ai-critic', active: true, role: 'session_page' },
        { tab_id: 2, index: 1, url: 'http://127.0.0.1:3080/', title: 'DSH', active: false, role: 'user' },
        { tab_id: 3, index: 2, url: 'http://127.0.0.1:8422/', title: 'Orchestrator', active: false, role: 'user' },
    ];
    assert.deepEqual(contentTabs(tabs).map((tab) => tab.tab_id), [2, 3]);
});

test('contentTabs tolerates no list and an all-plumbing list', () => {
    assert.deepEqual(contentTabs(null), []);
    assert.deepEqual(contentTabs(undefined), []);
    const page = { tab_id: 9, index: 0, url: 'http://127.0.0.1:43761/go?session=x', title: '', active: true, role: 'session_page' };
    assert.deepEqual(contentTabs([page]), []);
});

test('contentTabs keeps tabs from a daemon that does not send a role', () => {
    const tab = { tab_id: 4, index: 0, url: 'https://example.com/', title: 'Example', active: true };
    assert.deepEqual(contentTabs([tab]), [tab]);
});
