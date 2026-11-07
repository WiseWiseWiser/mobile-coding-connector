import { formatBytes } from '../../../api/processes';
import type { DiskSnapshot } from '../../../api/disk';
import { diskTone, formatCount, primaryVolume, usedPercent, volumeName } from './diskTone';

interface Props {
    disk: DiskSnapshot | null;
    error: string;
    onOpen: () => void;
}

/** Third, full-width tile on the System overview; opens the disk home. */
export default function DiskOverview({ disk, error, onOpen }: Props) {
    if (!disk && !error) return null;
    const vol = primaryVolume(disk);
    if (!vol && !error) return null;

    const tone = diskTone(vol);
    const pct = vol ? usedPercent(vol) : 0;
    const count = disk ? disk.volumes.length : 0;

    return (
        <button
            className="mcc-system-tile mcc-system-disk"
            onClick={onOpen}
            title="Open disk home"
        >
            <div className="mcc-system-tile-head">
                <span className="mcc-system-tile-lab">Disk</span>
                <span className="mcc-system-tile-more">
                    {error ? 'unavailable' : `${formatCount(count)} ›`}
                </span>
            </div>
            {vol ? (
                <>
                    <div className="mcc-system-disk-row">
                        <div className="mcc-system-tile-val">{formatBytes(vol.used_bytes)}</div>
                        <span className={`mcc-system-disk-pct ${tone}`}>{pct}%</span>
                    </div>
                    <div className={`mcc-system-bar ${tone}`}>
                        <i style={{ width: `${pct}%` }} />
                    </div>
                    <div className="mcc-system-tile-meta">
                        of {formatBytes(vol.total_bytes)} · {formatBytes(vol.free_bytes)} free ·{' '}
                        {volumeName(vol)} ({vol.fs_type.toUpperCase()})
                    </div>
                </>
            ) : (
                <>
                    <div className="mcc-system-disk-row">
                        <div className="mcc-system-tile-val">—</div>
                    </div>
                    <div className="mcc-system-tile-meta">capacity unavailable</div>
                </>
            )}
        </button>
    );
}
