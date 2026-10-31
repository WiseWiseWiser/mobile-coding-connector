import { centreView, clampView, scaleOf, screenToStream } from '../screen/viewport.ts';
import type { Point, ViewGeometry, Viewport } from '../screen/viewport.ts';
import type { BrowserTab } from './browserWire';

/** Anything rectangle-shaped: a DOMRect, or a plain object in tests. */
export interface RectLike {
    left: number;
    top: number;
    width: number;
    height: number;
}

/** The CSS transform that turns the fitted frame into the zoomed view. */
export interface ViewTransform {
    tx: number;
    ty: number;
    scale: number;
}

/**
 * The stage and the picture inside it, in stage-local CSS pixels.
 *
 * The <img> always fills the stage with `object-fit: contain`, so its box is the
 * stage and `fitOf` is the whole story: no letterbox arithmetic here.
 */
export function geometryOf(stage: RectLike, imgW: number, imgH: number): ViewGeometry {
    return { stageW: stage.width, stageH: stage.height, imgW, imgH };
}

/** Geometry for a stage that has not been measured yet (first paint). */
export function emptyGeometry(): ViewGeometry {
    return { stageW: 0, stageH: 0, imgW: 0, imgH: 0 };
}

/** True when geometry can be used: a stage box and a decoded frame. */
export function hasGeometry(g: ViewGeometry): boolean {
    return g.stageW > 0 && g.stageH > 0 && g.imgW > 0 && g.imgH > 0;
}

/** A pointer position in stage-local CSS pixels. */
export function localPoint(rect: RectLike, clientX: number, clientY: number): Point {
    return { x: clientX - rect.left, y: clientY - rect.top };
}

/**
 * Translate + scale for the frame element.
 *
 * The element is centred in the stage, so its centre already sits at the stage
 * centre; scaling about that centre moves the image point `(cx,cy)` to
 * `((cx-imgW/2)*k, (cy-imgH/2)*k)` away from the middle. The translate cancels
 * that, which puts `(cx,cy)` on the stage centre.
 */
export function transformOf(g: ViewGeometry, view: Viewport): ViewTransform {
    const settled = clampView(g, view);
    const k = scaleOf(g, settled);
    if (!k) return { tx: 0, ty: 0, scale: 1 };
    return {
        tx: -(settled.cx - g.imgW / 2) * k,
        ty: -(settled.cy - g.imgH / 2) * k,
        scale: settled.zoom,
    };
}

export function transformStyle(t: ViewTransform): string {
    if (t.tx === 0 && t.ty === 0 && t.scale === 1) return 'none';
    return `translate(${t.tx.toFixed(1)}px, ${t.ty.toFixed(1)}px) scale(${t.scale.toFixed(3)})`;
}

/**
 * A pointer in normalized page coordinates, [0,1].
 *
 * Inverse of the transform: undo the translate/scale with `screenToStream`, then
 * divide by the picture size. Taps in the black band around a letterboxed frame
 * clamp to the nearest edge — on a phone the frame is the whole stage, and a
 * near-miss should still click the control the human aimed at.
 */
export function normalizeAt(g: ViewGeometry, view: Viewport, local: Point): Point {
    if (!hasGeometry(g)) return { x: 0.5, y: 0.5 };
    const p = screenToStream(g, clampView(g, view), local);
    return {
        x: Math.min(1, Math.max(0, p.x / g.imgW)),
        y: Math.min(1, Math.max(0, p.y / g.imgH)),
    };
}

/** Normalized [0,1] back to the stage point where it paints. */
export function screenAt(g: ViewGeometry, view: Viewport, normalized: Point): Point {
    const settled = clampView(g, view);
    const k = scaleOf(g, settled);
    return {
        x: g.stageW / 2 + (normalized.x * g.imgW - settled.cx) * k,
        y: g.stageH / 2 + (normalized.y * g.imgH - settled.cy) * k,
    };
}

/** A fresh 1× view centred on the picture. */
export function resetView(g: ViewGeometry): Viewport {
    return hasGeometry(g) ? centreView(g) : { zoom: 1, cx: 0, cy: 0 };
}

/** Re-clamps after the stage or the frame size changed. */
export function settleView(g: ViewGeometry, view: Viewport): Viewport {
    return hasGeometry(g) ? clampView(g, view) : view;
}

/**
 * Every tab of a window, the /go page included. The rail is window-based: the
 * /go page is one of the window's tabs like any other, so lists and counts
 * must agree whether a window is previewed or currently driven.
 */
export function windowTabs(tabs: BrowserTab[] | null | undefined): BrowserTab[] {
    return tabs ?? [];
}

const PRESET_NAMES: Record<string, string> = {
    iphone: 'iPhone',
    ipad: 'iPad',
    mac: 'Mac',
};

/** Human name for a window preset, e.g. "iphone" -> "iPhone". */
export function presetLabel(name: string | null | undefined): string {
    if (!name) return 'Mac';
    return PRESET_NAMES[name] ?? name.charAt(0).toUpperCase() + name.slice(1);
}

/** The host part of a tab URL, without the scheme or path. */
export function tabHost(tab: BrowserTab): string {
    return (tab.url ?? '').replace(/^[a-z][a-z0-9+.-]*:\/\//i, '').split('/')[0] ?? '';
}

/** What a person calls this tab: its title, else its host, else a placeholder. */
export function tabLabel(tab: BrowserTab): string {
    const title = (tab.title ?? '').trim();
    if (title) return title;
    return tabHost(tab) || 'New tab';
}

/** One-character mark for a tab row. */
export function tabMark(tab: BrowserTab): string {
    return tabLabel(tab).charAt(0).toUpperCase();
}
