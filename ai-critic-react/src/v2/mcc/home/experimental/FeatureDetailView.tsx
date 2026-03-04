import { useState, useEffect } from 'react';
import { useNavigate, useParams } from 'react-router-dom';
import { BeakerIcon } from '../../../../pure-view/icons/BeakerIcon';
import { FeatureMakerContent } from '../../../../components/feature-maker';
import { useWorktreeRoute } from '../../../hooks/useWorktreeRoute';
import { fetchFeatures, updateFeature, type Feature } from '../../../../api/features';

export function FeatureDetailView() {
    const navigate = useNavigate();
    const { featureId } = useParams<{ featureId: string }>();
    const { projectName } = useWorktreeRoute();
    const [feature, setFeature] = useState<Feature | null>(null);
    const [editing, setEditing] = useState(false);
    const [editTitle, setEditTitle] = useState('');
    const [editDescription, setEditDescription] = useState('');

    useEffect(() => {
        if (!projectName || !featureId) return;
        fetchFeatures(projectName)
            .then(features => {
                const found = features.find(f => f.id === featureId);
                setFeature(found || null);
            })
            .catch(() => setFeature(null));
    }, [projectName, featureId]);

    const handleEdit = () => {
        setEditTitle(feature?.title ?? '');
        setEditDescription(feature?.description ?? '');
        setEditing(true);
    };

    const handleSave = () => {
        if (!projectName || !featureId) return;
        const updates = { title: editTitle, description: editDescription };
        updateFeature(projectName, featureId, updates)
            .then(updated => {
                setFeature(updated);
                setEditing(false);
            })
            .catch(() => {});
    };

    const handleCancel = () => {
        setEditing(false);
    };

    return (
        <div style={{ padding: '0 16px 16px' }}>
            <div className="mcc-section-header">
                <button className="mcc-back-btn" onClick={() => navigate('../feature-maker')}>
                    &larr;
                </button>
                <BeakerIcon className="mcc-header-icon" />
                <h2>Feature Maker</h2>
            </div>
            <FeatureMakerContent
                featureTitle={feature?.title ?? ''}
                featureDescription={feature?.description ?? ''}
                editing={editing}
                editTitle={editTitle}
                editDescription={editDescription}
                onEditTitleChange={setEditTitle}
                onEditDescriptionChange={setEditDescription}
                onEdit={handleEdit}
                onSave={handleSave}
                onCancel={handleCancel}
            />
        </div>
    );
}
