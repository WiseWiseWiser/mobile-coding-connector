import { formatBytes } from '../../../api/processes';
import type { CleanupPlan } from '../../../api/diskCleanup';
import SpaceCleanupRow from './SpaceCleanupRow';
import { classComposition, classSize } from './spaceCleanupTone';

interface Props {
    plan: CleanupPlan;
    selected: string[];
    pending: boolean;
    sizing: string[];
    onToggle: (id: string) => void;
    onSelectSafe: () => void;
}

/** The CLEANUP CLASSES block: one row per clear-tmp step, grouped by tier. */
export default function SpaceCleanupSection({ plan, selected, pending, sizing, onToggle, onSelectSafe }: Props) {
    const checked = new Set(selected);
    const readOnly = plan.classes.filter(c => c.readOnly).length;
    const clearable = plan.classes.length - readOnly;
    const groups = plan.groups.map(group => {
        const classes = plan.classes.filter(c => c.group === group.id);
        const bytes = classes.reduce((sum, c) => sum + (c.sizeBytes ?? 0), 0);
        return { group, classes, bytes };
    });

    return (
        <div className="mcc-cl-wrap">
            <div className="mcc-cl-head">
                <span className="mcc-cl-title">CLEANUP CLASSES</span>
                <span className="mcc-cl-count">
                    {clearable}
                    {readOnly > 0 ? ` + ${readOnly} read-only` : ''}
                </span>
                <button type="button" className="mcc-cl-all" onClick={onSelectSafe}>
                    select all safe
                </button>
            </div>

            {pending && (
                <div className="mcc-cl-pending">
                    measuring sizes in the background — rows fill in as each class is walked
                </div>
            )}

            {groups.map(({ group, classes, bytes }) => (
                <div key={group.id} className="mcc-cl-group">
                    <div className={`mcc-cl-tier ${group.id}`}>
                        <span>{group.label.toUpperCase()}</span>
                        <span className="mcc-cl-tier-note">{group.note}</span>
                    </div>
                    {classes.map(item => (
                        <SpaceCleanupRow
                            key={item.id}
                            item={item}
                            checked={checked.has(item.id)}
                            measuring={pending || sizing.includes(item.id)}
                            onToggle={onToggle}
                        />
                    ))}
                    <div className="mcc-cl-gfoot">
                        {classes.length} classes · {bytes > 0 ? formatBytes(bytes) : classSize(undefined)}
                    </div>
                </div>
            ))}

            <div className="mcc-cl-foot">
                classes come from clear-tmp · {classComposition(plan)}
            </div>
        </div>
    );
}
