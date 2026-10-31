import { useState } from 'react';
import { POLL_INTERVALS } from './browserPollStream';
import type { PollInterval } from './browserPollStream';
import type { ProbeEvidence } from './browserStream';
import type { StreamTransport } from './useBrowserStream';

export interface BrowserTransportProps {
    transport: StreamTransport;
    probe: ProbeEvidence | null;
    pollInterval: number;
    /** The probe banner shows once per episode; Retry re-arms the WebSocket. */
    onSwitchTransport: (next: StreamTransport) => void;
    onRetryWs: () => void;
    onPollInterval: (ms: PollInterval) => void;
}

function ms(value: number | null): string {
    return value === null ? 'no answer' : `${value} ms`;
}

/**
 * The poll-mode offer and controls. The banner appears only when the probe
 * caught a broken WebSocket on a reachable host; the badge in the address
 * banner and its menu are the persistent controls once poll mode runs.
 */
export function BrowserProbeBanner({ probe, onSwitchTransport, onRetryWs }: Pick<BrowserTransportProps, 'probe' | 'onSwitchTransport' | 'onRetryWs'>) {
    if (!probe) return null;
    return (
        <div className="bwp-probe" role="alertdialog" aria-label="Connection is slow">
            <div className="bwp-probe-h">⚠ Connection is slow</div>
            <p className="bwp-probe-b">
                Frames arrive over WebSocket, which seems blocked on this network. Your Mac is reachable — switch to poll
                mode to keep watching over plain HTTP.
            </p>
            <div className="bwp-probe-ev">
                <span>WebSocket <b className="is-no">{probe.reason === 'stalled' ? 'stalled' : `${probe.attempts} attempts failed`}</b></span>
                <span>REST api <b className={probe.restMs === null ? 'is-no' : 'is-ok'}>{ms(probe.restMs)}</b></span>
                {probe.frameAge >= 0 && <span>last frame <b>{Math.round(probe.frameAge / 1000)}s ago</b></span>}
            </div>
            <div className="bwp-probe-btns">
                <button type="button" className="bwp-probe-go" onClick={() => onSwitchTransport('poll')}>
                    Switch to Poll mode
                </button>
                <button type="button" className="bwp-probe-retry" onClick={onRetryWs}>
                    Retry
                </button>
            </div>
        </div>
    );
}

/** The pill in the address banner plus its interval / transport menu. */
export function BrowserPollBadge({ transport, pollInterval, onSwitchTransport, onPollInterval }: Pick<BrowserTransportProps, 'transport' | 'pollInterval' | 'onSwitchTransport' | 'onPollInterval'>) {
    const [open, setOpen] = useState(false);
    if (transport !== 'poll') return null;
    return (
        <>
            <button type="button" className="bwp-pill" onClick={() => setOpen((was) => !was)} aria-expanded={open} aria-label={`Poll mode, one frame per ${Math.round(pollInterval / 1000)} seconds`}>
                POLL {Math.round(pollInterval / 1000)}s
            </button>
            {open && (
                <div className="bwp-pillmenu" role="menu">
                    <div className="bwp-pillmenu-cap">Frame interval</div>
                    {POLL_INTERVALS.map((ms_) => (
                        <button
                            key={ms_}
                            type="button"
                            role="menuitemradio"
                            aria-checked={ms_ === pollInterval}
                            className="bwp-pillmenu-row"
                            onClick={() => {
                                onPollInterval(ms_);
                                setOpen(false);
                            }}
                        >
                            <span>{ms_ === 1000 ? 'Faster · battery cost' : ms_ === 3000 ? 'Balanced' : 'Slow network'}</span>
                            <b>{Math.round(ms_ / 1000)}s{ms_ === pollInterval ? ' ✓' : ''}</b>
                        </button>
                    ))}
                    <div className="bwp-pillmenu-cap">Transport</div>
                    <button
                        type="button"
                        role="menuitem"
                        className="bwp-pillmenu-row is-dim"
                        onClick={() => {
                            setOpen(false);
                            onSwitchTransport('ws');
                        }}
                    >
                        <span>Try WebSocket again →</span>
                    </button>
                    <div className="bwp-pillmenu-switchrow">
                        <span>Poll mode</span>
                        <button
                            type="button"
                            role="switch"
                            aria-checked
                            aria-label="Poll mode"
                            className="bwp-switch is-on"
                            onClick={() => {
                                setOpen(false);
                                onSwitchTransport('ws');
                            }}
                        >
                            <span aria-hidden="true" />
                        </button>
                    </div>
                </div>
            )}
        </>
    );
}
