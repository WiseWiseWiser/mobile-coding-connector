import { useEffect, useState } from 'react';
import { useNavigate } from 'react-router-dom';
import { BackIcon } from '../../../pure-view/icons/BackIcon';
import { fetchDisk, type DiskSnapshot } from '../../../api/disk';
import { formatBytes } from '../../../api/processes';
import DiskVolumeCard from './DiskVolumeCard';
import DiskMergedRow from './DiskMergedRow';
import DiskIOCard from './DiskIOCard';
import SpaceEntryCard from './SpaceEntryCard';
import {
    diskSubtitle,
    diskTone,
    primaryLabel,
    primaryVolume,
    sampledClock,
    usedPercent,
} from './diskTone';
import './SystemView.css';
import './DiskView.css';

const POLL_MS = 5000;

export default function DiskView() {
    const navigate = useNavigate();
    const [disk, setDisk] = useState<DiskSnapshot | null>(null);
    const [error, setError] = useState('');
    const [loading, setLoading] = useState(true);
    const [paused, setPaused] = useState(false);

    useEffect(() => {
        let cancelled = false;
        let timer: number | undefined;
        const tick = async () => {
            try {
                const next = await fetchDisk();
                if (!cancelled) {
                    setDisk(next);
                    setError('');
                }
            } catch (err) {
                if (!cancelled) setError(err instanceof Error ? err.message : String(err));
            } finally {
                if (!cancelled) setLoading(false);
            }
            if (!cancelled && !paused) timer = window.setTimeout(tick, POLL_MS);
        };
        tick();
        return () => {
            cancelled = true;
            if (timer) window.clearTimeout(timer);
        };
    }, [paused]);

    const vol = primaryVolume(disk);
    const tone = diskTone(vol);
    const pct = vol ? usedPercent(vol) : 0;
    const io = disk?.io;

    return (
        <div className="mcc-system">
            <div className="mcc-section-header" style={{ display: 'flex', alignItems: 'center', gap: 8 }}>
                <button
                    className="mcc-back-btn"
                    onClick={() => navigate('..', { relative: 'path' })}
                    style={{ background: 'none', border: 'none', cursor: 'pointer', padding: 4 }}
                >
                    <BackIcon />
                </button>
                <h2 style={{ margin: 0 }}>Disk</h2>
                <span className={`mcc-system-live${paused ? ' paused' : ''}`}>
                    {paused ? 'paused' : disk ? `${POLL_MS / 1000}s` : '…'}
                </span>
                <button
                    type="button"
                    className={`mcc-system-chip${paused ? ' on' : ''}`}
                    onClick={() => setPaused(v => !v)}
                >
                    {paused ? 'Resume' : 'Pause'}
                </button>
            </div>

            {disk && <div className="mcc-system-sub">{diskSubtitle(disk)}</div>}
            {loading && !disk && <div className="mcc-system-empty">Reading volumes…</div>}
            {!loading && !disk && (
                <div className="mcc-system-alert">
                    {error || 'capacity unavailable'} ·{' '}
                    {paused ? 'tap Resume to retry' : `retrying every ${POLL_MS / 1000}s`}
                </div>
            )}

            {disk && (
                <>
                    <div className="mcc-system-tile mcc-system-disk">
                        <div className="mcc-system-tile-head">
                            <span className="mcc-system-tile-lab">{vol ? primaryLabel(vol) : 'Disk'}</span>
                            <span className="mcc-system-tile-more">{vol ? vol.fs_type.toUpperCase() : '—'}</span>
                        </div>
                        <div className="mcc-system-disk-row">
                            <div className="mcc-system-tile-val">
                                {vol ? `${formatBytes(vol.used_bytes)} used` : '—'}
                            </div>
                            {vol && <span className={`mcc-system-disk-pct ${tone}`}>{pct}%</span>}
                        </div>
                        {vol && (
                            <div className={`mcc-system-bar ${tone}`}>
                                <i style={{ width: `${pct}%` }} />
                            </div>
                        )}
                        <div className="mcc-system-tile-meta">
                            {vol
                                ? `free ${formatBytes(vol.free_bytes)} of ${formatBytes(vol.total_bytes)}${sampledClock(disk.sampled_at) ? ` · sampled ${sampledClock(disk.sampled_at)}` : ''}`
                                : 'capacity unavailable'}
                        </div>
                    </div>

                    <SpaceEntryCard />

                    <div className="mcc-disk-sec">
                        <h3>Volumes</h3>
                        <span>{disk.volumes.length} mounted</span>
                    </div>
                    {disk.volumes.length === 0 && (
                        <div className="mcc-system-empty">No volumes could be read</div>
                    )}
                    {disk.volumes.map(v => (
                        <DiskVolumeCard key={`${v.device}:${v.mount}`} vol={v} />
                    ))}
                    {(disk.merged ?? []).map(m => (
                        <DiskMergedRow key={`merged:${m.mount}`} entry={m} />
                    ))}

                    {io ? <DiskIOCard io={io} device={vol?.device ?? disk.primary_mount} /> : null}

                    {disk.warning && <div className="mcc-system-alert">{disk.warning}</div>}

                    <div className="mcc-disk-foot">
                        GET /api/server/disk · {POLL_MS / 1000}s · statfs{io ? ' + hostmon (linux)' : ''}
                    </div>
                </>
            )}

            {error && disk && (
                <div className="mcc-system-alert">refresh failed · showing the last sample</div>
            )}
        </div>
    );
}
