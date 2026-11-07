import { formatBytes } from '../../../api/processes';
import type { AnalysisReport } from '../../../api/diskAnalysis';

/** Stable per-category colours; ids come from the server and never change. */
const CATEGORY_COLORS: Record<string, string> = {
    xcode: '#60a5fa',
    'dev-cache': '#22d3ee',
    containers: '#a78bfa',
    caches: '#38bdf8',
    media: '#f472b6',
    downloads: '#fbbf24',
    logs: '#f87171',
    other: '#475569',
};

export function categoryColor(id: string): string {
    return CATEGORY_COLORS[id] ?? CATEGORY_COLORS.other;
}

export function isStale(iso: string | undefined, hours = 24): boolean {
    if (!iso) return true;
    const at = new Date(iso);
    if (Number.isNaN(at.getTime())) return true;
    return Date.now() - at.getTime() > hours * 3600 * 1000;
}

/** "just now", "8 min ago", "3 h ago", "2 days ago". */
export function relativeTime(iso: string | undefined): string {
    if (!iso) return 'never';
    const at = new Date(iso);
    if (Number.isNaN(at.getTime())) return 'never';
    const sec = Math.max(0, Math.round((Date.now() - at.getTime()) / 1000));
    if (sec < 45) return 'just now';
    const min = Math.round(sec / 60);
    if (min < 60) return `${min} min ago`;
    const hours = Math.round(min / 60);
    if (hours < 24) return `${hours} h ago`;
    const days = Math.round(hours / 24);
    if (days < 30) return `${days} ${days === 1 ? 'day' : 'days'} ago`;
    return shortWhen(iso);
}

/** "11-05 09:12" — no year, newest-first lists do not need it. */
export function shortWhen(iso: string | undefined): string {
    if (!iso) return '—';
    const at = new Date(iso);
    if (Number.isNaN(at.getTime())) return '—';
    const pad = (n: number) => String(n).padStart(2, '0');
    return `${pad(at.getMonth() + 1)}-${pad(at.getDate())} ${pad(at.getHours())}:${pad(at.getMinutes())}`;
}

export function clockTime(iso: string | undefined): string {
    if (!iso) return '';
    const at = new Date(iso);
    if (Number.isNaN(at.getTime())) return '';
    return at.toLocaleTimeString([], { hour12: false });
}

export function duration(ms: number): string {
    if (!ms || ms < 0) return '—';
    if (ms < 1000) return `${ms} ms`;
    return `${(ms / 1000).toFixed(ms < 10000 ? 1 : 0)} s`;
}

export function fileCount(files: number): string {
    if (!files) return '0 files';
    if (files >= 1_000_000) return `${(files / 1_000_000).toFixed(1)}M files`;
    if (files >= 1000) return `${Math.round(files / 1000)}k files`;
    return `${files} files`;
}

/** Growth is bad: +3 GB is amber, −9 GB is green, "same" is neutral. */
export function signedBytes(delta: number): string {
    if (!Number.isFinite(delta) || delta === 0) return 'same';
    const sign = delta > 0 ? '+' : '−';
    return `${sign}${formatBytes(Math.abs(delta))}`;
}

export function deltaTone(delta: number): 'up' | 'down' | 'flat' {
    if (!Number.isFinite(delta) || delta === 0) return 'flat';
    return delta > 0 ? 'up' : 'down';
}

/** A stored report and its summary share the category rollup, so both feed these helpers. */
type Rollup = Pick<AnalysisReport, 'categories'> & { reclaimable?: number };

export function reclaimableTotal(summary: Rollup | null | undefined): number {
    if (!summary) return 0;
    if (summary.reclaimable && summary.reclaimable > 0) return summary.reclaimable;
    return summary.categories.reduce((sum, c) => sum + (c.reclaimable || 0), 0);
}

export function categoryBytes(summary: Rollup | null | undefined, id: string): number | null {
    if (!summary) return null;
    const found = summary.categories.find(c => c.id === id);
    return found ? found.bytes : null;
}

export function percentOf(part: number, total: number): number {
    if (!total || total <= 0) return 0;
    return Math.max(0, Math.min(100, Math.round((part / total) * 100)));
}

/** Categories of a report, biggest first; empty buckets are dropped. */
export function sortedCategories(summary: Rollup | null | undefined) {
    if (!summary) return [];
    return summary.categories
        .filter(c => c.bytes > 0)
        .slice()
        .sort((a, b) => b.bytes - a.bytes);
}
