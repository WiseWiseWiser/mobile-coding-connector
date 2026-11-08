import type { CleanupClass } from '../../../api/diskCleanup';
import { actionBadge, classLine, classSize, classTag } from './spaceCleanupTone';

interface Props {
    item: CleanupClass;
    checked: boolean;
    onToggle: (id: string) => void;
    /** Preview rows only show what would run; they are not interactive. */
    preview?: boolean;
    /** True while a measurement pass is in flight; sizes not yet known. */
    measuring?: boolean;
}

/** One cleanup class: checkbox, name + tag, the command line, its size. */
export default function SpaceCleanupRow({ item, checked, onToggle, preview, measuring }: Props) {
    const tag = classTag(item);
    const action = item.actions?.[0];
    const kind = action?.kind ?? 'none';
    const locked = item.readOnly;
    const hint = item.notes && item.notes.length > 0 ? item.notes[0] : '';
    const boxClass = locked ? ' dash' : checked ? ' on' : '';

    return (
        <button
            type="button"
            className={`mcc-cl${checked ? ' sel' : ''}${locked ? ' dis' : ''}`}
            onClick={() => {
                if (!locked && !preview) onToggle(item.id);
            }}
            disabled={locked || preview}
        >
            <span className={`mcc-cl-box${boxClass}`} />
            <div className="mcc-cl-main">
                <div className="mcc-cl-name">
                    {item.label}
                    {tag && <span className={`mcc-cl-tag ${tag.replace(/[^a-z]/g, '')}`}>{tag}</span>}
                </div>
                <div className="mcc-cl-cmd">
                    {kind !== 'none' && (
                        <span className={`mcc-cl-badge ${kind}`}>{actionBadge(kind)}</span>
                    )}
                    <span className="mcc-cl-line" title={classLine(item)}>
                        {classLine(item)}
                    </span>
                </div>
                {hint && <div className="mcc-cl-hint">{hint}</div>}
            </div>
            <span className="mcc-cl-size">
                {classSize(item.sizeBytes)}
                {!item.measured && <small>{measuring ? 'measuring' : 'not sized'}</small>}
            </span>
        </button>
    );
}
