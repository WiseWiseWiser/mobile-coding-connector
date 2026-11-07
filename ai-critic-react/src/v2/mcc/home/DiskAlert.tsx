import { formatBytes } from '../../../api/processes';
import type { DiskSnapshot } from '../../../api/disk';
import { diskTone, primaryLabel, primaryVolume, usedPercent } from './diskTone';

/** Shown on the overview once a volume gets dangerously full. */
export default function DiskAlert({ disk }: { disk: DiskSnapshot | null }) {
    const vol = primaryVolume(disk);
    if (!vol || diskTone(vol) !== 'hot') return null;
    return (
        <div className="mcc-system-alert bad">
            {primaryLabel(vol)} is {usedPercent(vol)}% full · {formatBytes(vol.free_bytes)} free.
            Free space or move large files before writes start failing.
        </div>
    );
}
