import type { DiskSnapshot } from '../../../api/disk';
import { formatRate } from './diskTone';

/** One-line disk KPI, e.g. "Disk R 12.4 · W 3.4 MB/s". */
export default function DiskIOKpi({ disk }: { disk: DiskSnapshot | null }) {
    const io = disk?.io;
    if (!io) return null;
    return (
        <span>
            Disk{' '}
            <b>
                R {formatRate(io.read_bps)} · W {formatRate(io.write_bps)}
            </b>
        </span>
    );
}
