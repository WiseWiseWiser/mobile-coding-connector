import { useEffect, useRef } from 'react';
import { START_STEPS, type LogLine } from './wsProxyStream';

function StepBar({ current, failed }: { current: string | null; failed: boolean }) {
    if (!current) return null;
    const cur = START_STEPS.indexOf(current);
    return (
        <div className="mcc-steps">
            {START_STEPS.map((s) => {
                const at = START_STEPS.indexOf(s);
                let cls = '';
                if (at < cur) cls = 'done';
                else if (at === cur) cls = failed ? 'fail' : 'active';
                const mark = at < cur ? ' ✓' : at === cur && !failed ? ' …' : at === cur && failed ? ' ✗' : '';
                return (
                    <span key={s} className={`mcc-step ${cls}`}>{s}{mark}</span>
                );
            })}
        </div>
    );
}

export function WSProxyLogArea({
    id,
    lines,
    streaming,
    onClear,
    onHide,
    step,
    failed,
}: {
    id: string;
    lines: LogLine[];
    streaming: boolean;
    onClear: () => void;
    onHide: () => void;
    step: string | null;
    failed: boolean;
}) {
    const bodyRef = useRef<HTMLDivElement>(null);
    useEffect(() => {
        if (bodyRef.current) bodyRef.current.scrollTop = bodyRef.current.scrollHeight;
    }, [lines]);
    return (
        <div className="mcc-sys-log">
            <div className="mcc-sys-log-head">
                <span className={`mcc-sys-log-dot ${streaming ? 'on' : ''}`} />
                <span>{id} · {streaming ? 'operation' : 'log'}</span>
                <span className="mcc-sys-log-spacer" />
                <button type="button" onClick={onClear}>Clear</button>
                <button type="button" onClick={onHide}>Hide</button>
            </div>
            <StepBar current={step} failed={failed} />
            <div className="mcc-sys-log-body" ref={bodyRef}>
                {lines.length === 0 && (
                    <div className="mcc-sys-log-empty">点击 Start 后,操作日志将在这里实时输出…</div>
                )}
                {lines.map((l, i) => (
                    <div key={i} className={l.error ? 'mcc-sys-log-line err' : 'mcc-sys-log-line'}>{l.text}</div>
                ))}
            </div>
        </div>
    );
}
