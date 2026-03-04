import './FeatureRequestSection.css';

interface FeatureRequestSectionProps {
    featureTitle?: string;
    featureDescription?: string;
    editing?: boolean;
    editTitle?: string;
    editDescription?: string;
    onEditTitleChange?: (title: string) => void;
    onEditDescriptionChange?: (description: string) => void;
    onEdit?: () => void;
    onSave?: () => void;
    onCancel?: () => void;
}

export function FeatureRequestSection({
    featureTitle,
    featureDescription,
    editing,
    editTitle,
    editDescription,
    onEditTitleChange,
    onEditDescriptionChange,
    onEdit,
    onSave,
    onCancel,
}: FeatureRequestSectionProps) {
    return (
        <div className="fm-feature-request">
            <div className="fm-fr-header">
                <h3>Feature Request</h3>
                {!editing && (
                    <button className="fm-fr-edit-icon-btn" onClick={onEdit} title="Edit">
                        <svg width="16" height="16" viewBox="0 0 16 16" fill="currentColor">
                            <path d="M11.13 1.47a1.5 1.5 0 0 1 2.12 0l1.28 1.28a1.5 1.5 0 0 1 0 2.12l-8.5 8.5a.75.75 0 0 1-.35.2l-3.5 1a.75.75 0 0 1-.92-.92l1-3.5a.75.75 0 0 1 .2-.35l8.67-8.33ZM12.19 2.53l-8.33 8.33-.6 2.07 2.07-.6 8.33-8.33-1.47-1.47Z" />
                        </svg>
                    </button>
                )}
            </div>
            <div className="fm-fr-content">
                {editing ? (
                    <>
                        <input
                            className="fm-fr-title-input"
                            value={editTitle ?? ''}
                            onChange={e => onEditTitleChange?.(e.target.value)}
                            placeholder="Feature title"
                        />
                        <textarea
                            className="fm-fr-desc-input"
                            value={editDescription ?? ''}
                            onChange={e => onEditDescriptionChange?.(e.target.value)}
                            placeholder="Feature description (optional)"
                            rows={3}
                        />
                        <div className="fm-fr-edit-actions">
                            <button className="fm-fr-save-btn" onClick={onSave}>Save</button>
                            <button className="fm-fr-cancel-btn" onClick={onCancel}>Cancel</button>
                        </div>
                    </>
                ) : (
                    <>
                        <h4 className="fm-fr-view-title">{featureTitle || '(Untitled)'}</h4>
                        <p className="fm-fr-view-desc">{featureDescription || '(No description)'}</p>
                    </>
                )}
            </div>
        </div>
    );
}
