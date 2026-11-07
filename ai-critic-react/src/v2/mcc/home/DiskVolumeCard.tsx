import { formatBytes } from '../../../api/processes';
import type { DiskVolume } from '../../../api/disk';
import { diskTone, usedPercent, volumeName } from './diskTone';

/** One row of the volume list on the disk home. */
export default function DiskVolumeCard({ vol }: { vol: DiskVolume }) {
    const tone = diskTone(vol);
    const pct = usedPercent(vol);
    return (
        <div className={`mcc-disk-vol${vol.primary ? ' primary' : ''}`}>
            <div className="mcc-disk-vol-top">
                <span className="mcc-disk-vol-name">{volumeName(vol)}</span>
                <span className="mcc-disk-vol-fs">{vol.fs_type.toUpperCase()}</span>
                {vol.read_only && <span className="mcc-disk-vol-ro">read-only</span>}
                <span className="mcc-disk-vol-free">{formatBytes(vol.free_bytes)} free</span>
            </div>
            <div className="mcc-disk-vol-size">
                {formatBytes(vol.used_bytes)} <span>of {formatBytes(vol.total_bytes)} · {pct}%</span>
            </div>
            <div className={`mcc-system-bar ${tone}`}>
                <i style={{ width: `${pct}%` }} />
            </div>
            <div className="mcc-disk-vol-path">{vol.mount}</div>
        </div>
    );
}
