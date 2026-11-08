import type { CleanupClass, CleanupPlan } from '../../../api/diskCleanup';

export type CleanupRowStatus = 'running' | 'done' | 'warn' | 'skipped';

export interface CleanupRunRow {
    id: string;
    name: string;
    status: CleanupRowStatus;
    detail: string;
}

/** The tier a class belongs to, so the panel can order and label it. */
export function cleanupGroupOf(plan: CleanupPlan | null, group: string): { label: string; note: string } {
    const found = plan?.groups.find(g => g.id === group);
    return { label: found?.label ?? group, note: found?.note ?? '' };
}

/** Short tag shown after the class name. */
export function classTag(c: CleanupClass): string {
    if (c.readOnly) return c.group === 'inventory' ? 'never' : 'read-only';
    if (c.needsAdmin) return 'admin';
    if (c.group === 'aggressive') return 'added';
    return '';
}

/** The badge on the command line: how this class is actually cleared. */
export function actionBadge(kind: string): string {
    if (kind === 'rm-rf') return 'rm -rf';
    if (kind === 'official') return 'official';
    if (kind === 'sudo') return 'sudo';
    return 'none';
}

/** Path or command to show under the class name. */
export function classLine(c: CleanupClass): string {
    const action = c.actions?.[0];
    if (action && action.text) return action.text;
    if (c.paths && c.paths.length > 0) return c.paths.join(' · ');
    return c.command || '';
}

export function classSize(size?: number): string {
    if (size === undefined || size === null) return '—';
    return formatCleanupBytes(size);
}

/** Same unit rules as the rest of the disk views. */
export function formatCleanupBytes(bytes: number): string {
    if (!bytes) return '0 B';
    const units = ['B', 'KB', 'MB', 'GB', 'TB'];
    let n = bytes;
    let i = 0;
    while (n >= 1024 && i < units.length - 1) {
        n /= 1024;
        i += 1;
    }
    const digits = n >= 10 || i === 0 ? 0 : 1;
    return `${n.toFixed(digits)} ${units[i]}`;
}

export function sumSelected(plan: CleanupPlan | null, selected: string[]): number {
    if (!plan) return 0;
    const ids = new Set(selected);
    return plan.classes.reduce((sum, c) => (ids.has(c.id) ? sum + (c.sizeBytes ?? 0) : sum), 0);
}

export function unmeasuredSelected(plan: CleanupPlan | null, selected: string[]): number {
    if (!plan) return 0;
    const ids = new Set(selected);
    return plan.classes.filter(c => ids.has(c.id) && c.sizeBytes === undefined).length;
}

export function defaultSelection(plan: CleanupPlan | null): string[] {
    if (!plan) return [];
    return plan.classes.filter(c => c.selected && !c.readOnly).map(c => c.id);
}

/** `8 default + 8 aggressive + 3 system + 2 read-only` — the composition line. */
export function classComposition(plan: CleanupPlan | null): string {
    if (!plan) return '';
    const parts = plan.groups.map(g => {
        const count = plan.classes.filter(c => c.group === g.id).length;
        return `${count} ${g.label.toLowerCase()}`;
    });
    return parts.filter(Boolean).join(' + ');
}

export function statusGlyph(status: CleanupRowStatus): string {
    if (status === 'done') return '✓';
    if (status === 'warn') return '!';
    if (status === 'skipped') return '–';
    return '•';
}
