import assert from 'node:assert/strict';
import { test } from 'node:test';
import {
    DOUBLE_TAP_ZOOM,
    MAX_ZOOM,
    MIN_ZOOM,
    ZOOM_STEP,
    centreView,
    clampView,
    clampZoom,
    drawRectOf,
    fitOf,
    isZoomed,
    panBy,
    scaleOf,
    screenToImage,
    screenToStream,
    stepZoom,
    streamToScreen,
    toggleZoom,
    zoomAt,
} from '../src/v2/mcc/screen/viewport.ts';

// iPhone-ish stage, MacBook-ish stream: the same numbers the mirror sees.
const stage = { stageW: 390, stageH: 660, imgW: 1280, imgH: 826 };
const centre = centreView(stage);

const near = (actual, expected, tolerance = 1e-6) =>
    assert.ok(
        Math.abs(actual - expected) <= tolerance,
        `expected ${actual} to be within ${tolerance} of ${expected}`,
    );

test('fit letterboxes a wide image into a tall phone stage', () => {
    const fit = fitOf(stage);
    near(fit.scale, 390 / 1280);
    near(fit.drawW, 390);
    near(fit.drawH, (826 * 390) / 1280);
    near(fit.offX, 0);
    near(fit.offY, (660 - fit.drawH) / 2);
    // drawH 251.6 and offY 204.2, i.e. the 204/252 the mockup showed.
    near(Math.round(fit.drawH), 252);
    near(Math.round(fit.offY), 204);
});

test('1x paints exactly the fit rect — no behaviour change to today', () => {
    const fit = fitOf(stage);
    const rect = drawRectOf(stage, centre);
    near(rect.sx, 0);
    near(rect.sy, 0);
    near(rect.sw, stage.imgW);
    near(rect.sh, stage.imgH);
    near(rect.dx, fit.offX);
    near(rect.dy, fit.offY);
    near(rect.dw, fit.drawW);
    near(rect.dh, fit.drawH);
    assert.equal(isZoomed(centre), false);
});

test('screen <-> stream round trips at every zoom level', () => {
    for (const zoom of [MIN_ZOOM, 2.4, 6]) {
        const view = clampView(stage, { zoom, cx: 700, cy: 500 });
        for (const point of [
            { x: 0, y: 0 },
            { x: 195, y: 330 },
            { x: 389, y: 659 },
        ]) {
            const back = streamToScreen(stage, view, screenToStream(stage, view, point));
            near(back.x, point.x, 1e-6);
            near(back.y, point.y, 1e-6);
        }
    }
    near(scaleOf(stage, centre), fitOf(stage).scale);
});

test('pinch keeps the anchored pixel pinned to the same spot', () => {
    // Both zoom levels cover the stage, so nothing has to be clamped.
    const anchor = { x: 900, y: 420 };
    for (const [from, to] of [
        [3, 4.5],
        [4.5, 3],
        [3, 6],
    ]) {
        const before = clampView(stage, { zoom: from, cx: 640, cy: 420 });
        const after = zoomAt(stage, before, to, anchor);
        near(streamToScreen(stage, before, anchor).x, streamToScreen(stage, after, anchor).x, 1e-6);
        near(streamToScreen(stage, before, anchor).y, streamToScreen(stage, after, anchor).y, 1e-6);
        near(after.zoom, to);
    }
});

test('zoom is clamped to 1x..6x', () => {
    assert.equal(clampZoom(0.1), MIN_ZOOM);
    assert.equal(clampZoom(99), MAX_ZOOM);
    assert.equal(clampZoom(Number.NaN), MIN_ZOOM);
    assert.equal(zoomAt(stage, centre, 99, { x: 10, y: 10 }).zoom, MAX_ZOOM);
    assert.equal(zoomAt(stage, centre, -3, { x: 10, y: 10 }).zoom, MIN_ZOOM);
});

test('at full zoom the image always covers the stage', () => {
    const zoom = MAX_ZOOM;
    const k = scaleOf(stage, { zoom, cx: 0, cy: 0 });
    const halfW = stage.stageW / (2 * k);
    const halfH = stage.stageH / (2 * k);
    assert.ok(halfW * 2 < stage.imgW, 'image must be wider than the stage at 6x');
    for (const cx of [-500, halfW, 400, stage.imgW - halfW, 5000]) {
        for (const cy of [-500, halfH, 300, stage.imgH - halfH, 5000]) {
            const rect = drawRectOf(stage, { zoom, cx, cy });
            assert.ok(rect.dx <= 1e-6, `left band at cx=${cx}: dx=${rect.dx}`);
            assert.ok(rect.dy <= 1e-6, `top band at cy=${cy}: dy=${rect.dy}`);
            assert.ok(rect.dx + rect.dw >= stage.stageW - 1e-6, 'right band');
            assert.ok(rect.dy + rect.dh >= stage.stageH - 1e-6, 'bottom band');
        }
    }
});

test('the centre cannot be dragged past the edge of the image', () => {
    const view = clampView(stage, { zoom: MAX_ZOOM, cx: -9999, cy: 9999 });
    const k = scaleOf(stage, view);
    near(view.cx, stage.stageW / (2 * k));
    near(view.cy, stage.imgH - stage.stageH / (2 * k));
    const clamped = clampView(stage, { zoom: MAX_ZOOM, cx: 9999, cy: 0 });
    near(clamped.cx, stage.imgW - stage.stageW / (2 * k));
    near(clamped.cy, stage.stageH / (2 * k));
});

test('an axis the image cannot fill stays centred', () => {
    // At 2x the image is wider than the stage but still shorter than the stage.
    const view = clampView(stage, { zoom: 2, cx: 400, cy: 0 });
    near(view.cy, stage.imgH / 2);
    near(view.cx, 400);
    assert.equal(isZoomed(view), true);
    // Horizontal panning has a floor: the image edge may not lift off the stage.
    near(clampView(stage, { zoom: 2, cx: 10, cy: 0 }).cx, stage.stageW / (2 * scaleOf(stage, view)));
    // At 1x there is nothing to pan at all.
    const panned = panBy(stage, centre, 120, -80);
    near(panned.cx, stage.imgW / 2);
    near(panned.cy, stage.imgH / 2);
    near(panned.zoom, MIN_ZOOM);
});

test('dragging moves the picture with the finger', () => {
    const view = clampView(stage, { zoom: 3, cx: 640, cy: 420 });
    const probe = { x: 500, y: 400 };
    const before = streamToScreen(stage, view, probe);
    const panned = panBy(stage, view, 60, 40);
    const after = streamToScreen(stage, panned, probe);
    near(after.x - before.x, 60, 1e-6);
    near(after.y - before.y, 40, 1e-6);
});

test('rotating the phone keeps the centre of interest on the same pixel', () => {
    const landscape = { stageW: 660, stageH: 390, imgW: 1280, imgH: 826 };
    const view = clampView(stage, { zoom: 3.5, cx: 800, cy: 500 });
    const rotated = clampView(landscape, view);
    near(rotated.cx, 800);
    near(rotated.cy, 500);
    near(rotated.zoom, 3.5);
});

test('taps map to image pixels and ignore the letterbox band', () => {
    const hero = screenToImage(stage, centre, { x: stage.stageW / 2, y: stage.stageH / 2 });
    assert.ok(hero);
    near(hero.x, stage.imgW / 2, 1);
    near(hero.y, stage.imgH / 2, 1);
    // Top of the stage is above the image, so the tap belongs to nobody.
    assert.equal(screenToImage(stage, centre, { x: 195, y: 10 }), null);
    // Once zoomed the band is gone and corner taps hit real pixels.
    const zoomed = clampView(stage, { zoom: 6, cx: stage.imgW / 2, cy: stage.imgH / 2 });
    const corner = screenToImage(stage, zoomed, { x: 1, y: 1 });
    assert.ok(corner);
    assert.ok(corner.x < stage.imgW / 2 && corner.y < stage.imgH / 2);
});

test('zoom steps and double tap stay inside the range', () => {
    near(stepZoom(1, 1), ZOOM_STEP);
    near(stepZoom(ZOOM_STEP, -1), 1);
    assert.equal(stepZoom(MAX_ZOOM, 1), MAX_ZOOM);
    assert.equal(stepZoom(MIN_ZOOM, -1), MIN_ZOOM);
    assert.equal(toggleZoom(MIN_ZOOM), DOUBLE_TAP_ZOOM);
    assert.equal(toggleZoom(DOUBLE_TAP_ZOOM), MIN_ZOOM);
    assert.equal(toggleZoom(6), MIN_ZOOM);
});

test('a detached or empty stage is inert instead of throwing', () => {
    const empty = { stageW: 0, stageH: 0, imgW: 0, imgH: 0 };
    assert.deepEqual(fitOf(empty), { scale: 0, drawW: 0, drawH: 0, offX: 0, offY: 0 });
    const rect = drawRectOf(empty, { zoom: 3, cx: 0, cy: 0 });
    assert.deepEqual(rect, { sx: 0, sy: 0, sw: 0, sh: 0, dx: 0, dy: 0, dw: 0, dh: 0 });
    assert.equal(screenToImage(empty, { zoom: 3, cx: 0, cy: 0 }, { x: 1, y: 1 }), null);
    assert.deepEqual(screenToStream(empty, { zoom: 3, cx: 0, cy: 0 }, { x: 1, y: 1 }), { x: 0, y: 0 });
});
