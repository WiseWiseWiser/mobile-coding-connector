import type { DiskIO } from '../../../api/disk';
import { formatRate, sampledClock } from './diskTone';

/** 15s hostmon average, Linux only; hidden entirely when absent. */
export default function DiskIOCard({ io, device }: { io: DiskIO; device: string }) {
    const clock = sampledClock(io.sampled_at);
    return (
        <>
            <div className="mcc-disk-sec">
                <h3>I/O</h3>
                <span>{io.interval_sec}s avg · hostmon</span>
            </div>
            <div className="mcc-disk-io">
                <div className="mcc-disk-io-cell">
                    <div className="mcc-disk-io-lab">Read</div>
                    <div className="mcc-disk-io-val">{formatRate(io.read_bps)}</div>
                    <div className="mcc-disk-io-meta">
                        {io.read_iops} iops · {io.interval_sec}s
                    </div>
                </div>
                <div className="mcc-disk-io-cell">
                    <div className="mcc-disk-io-lab">Write</div>
                    <div className="mcc-disk-io-val">{formatRate(io.write_bps)}</div>
                    <div className="mcc-disk-io-meta">
                        {io.write_iops} iops · {io.interval_sec}s
                    </div>
                </div>
            </div>
            <div className="mcc-disk-foot">
                {io.device || device} · sampled {clock || io.sampled_at}
            </div>
        </>
    );
}
