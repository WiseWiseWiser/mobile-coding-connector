import { formatBytes } from '../../../api/processes';
import type { DiskSnapshot, DiskVolume } from '../../../api/disk';

export type DiskTone = '' | 'warn' | 'hot';

export const DATA_MOUNT = '/System/Volumes/Data';

const GIB = 1024 * 1024 * 1024;

export function primaryVolume(disk: DiskSnapshot | null): DiskVolume | null {
    if (!disk || disk.volumes.length === 0) return null;
    return disk.volumes.find(v => v.primary) ?? disk.volumes[0];
}

/** Blue below 80%, amber from 80%, red from 95% or under 10 GB free. */
export function diskTone(vol: DiskVolume | null | undefined): DiskTone {
    if (!vol) return '';
    if (vol.used_percent >= 95 || (vol.total_bytes > 0 && vol.free_bytes < 10 * GIB)) return 'hot';
    if (vol.used_percent >= 80) return 'warn';
    return '';
}

export function usedPercent(vol: DiskVolume): number {
    const pct = Math.round(vol.used_percent);
    if (!Number.isFinite(pct)) return 0;
    return Math.max(0, Math.min(100, pct));
}

export function volumeName(vol: DiskVolume): string {
    if (vol.mount === DATA_MOUNT) return 'Data';
    if (vol.mount === '/') return 'Root';
    return mountName(vol.mount);
}

/** Hero label of the boot volume, e.g. "Boot volume · Data". */
export function primaryLabel(vol: DiskVolume): string {
    if (vol.mount === DATA_MOUNT) return 'Boot volume · Data';
    if (vol.mount === '/') return 'Boot volume';
    return `Volume · ${mountName(vol.mount)}`;
}

export function mountName(mount: string): string {
    const parts = mount.split('/').filter(Boolean);
    return parts.length ? parts[parts.length - 1] : mount;
}

export function totalBytes(disk: DiskSnapshot | null): number {
    if (!disk) return 0;
    return disk.volumes.reduce((sum, v) => sum + v.total_bytes, 0);
}

export function formatRate(bps: number): string {
    const units = ['B', 'KB', 'MB', 'GB', 'TB'];
    let n = bps || 0;
    let i = 0;
    while (n >= 1024 && i < units.length - 1) {
        n /= 1024;
        i++;
    }
    const digits = i === 0 || n >= 100 ? 0 : 1;
    return `${n.toFixed(digits)} ${units[i]}/s`;
}

export function formatCount(n: number): string {
    return `${n} ${n === 1 ? 'volume' : 'volumes'}`;
}

export function diskSubtitle(disk: DiskSnapshot | null): string {
    if (!disk || disk.volumes.length === 0) return 'No volumes detected';
    return `${formatCount(disk.volumes.length)} · ${formatBytes(totalBytes(disk))} total · host of this page`;
}

export function sampledClock(sampledAt: string): string {
    const d = new Date(sampledAt);
    if (!sampledAt || Number.isNaN(d.getTime())) return '';
    return d.toLocaleTimeString([], { hour12: false });
}
