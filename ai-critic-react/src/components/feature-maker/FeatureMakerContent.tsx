import type { FeatureMakerContentProps } from './types';
import { FeatureRequestSection } from './FeatureRequestSection';
import './FeatureMakerContent.css';

export function FeatureMakerContent({
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
}: FeatureMakerContentProps = {}) {
    return (
        <div className="feature-maker">
            <div className="fm-content">
                <FeatureRequestSection
                    featureTitle={featureTitle}
                    featureDescription={featureDescription}
                    editing={editing}
                    editTitle={editTitle}
                    editDescription={editDescription}
                    onEditTitleChange={onEditTitleChange}
                    onEditDescriptionChange={onEditDescriptionChange}
                    onEdit={onEdit}
                    onSave={onSave}
                    onCancel={onCancel}
                />
            </div>
        </div>
    );
}
