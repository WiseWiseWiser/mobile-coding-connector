import type { DiskMergedMount } from '../../../api/disk';
import { mountName } from './diskTone';

/** A read-only APFS companion mount that is already counted in another volume. */
export default function DiskMergedRow({ entry }: { entry: DiskMergedMount }) {
    const name = entry.mount === '/' ? 'Root snapshot' : mountName(entry.mount);
    return (
        <div className="mcc-disk-vol merged">
            <div className="mcc-disk-vol-top">
                <span className="mcc-disk-vol-name">{name}</span>
                <span className="mcc-disk-vol-fs">{entry.fs_type.toUpperCase()}</span>
                <span className="mcc-disk-vol-free">merged</span>
            </div>
            <div className="mcc-disk-vol-path">
                {entry.mount} · {entry.reason}
            </div>
        </div>
    );
}
