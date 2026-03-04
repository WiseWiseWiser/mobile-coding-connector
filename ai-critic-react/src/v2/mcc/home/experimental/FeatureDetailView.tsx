import { useState, useEffect, useRef, useCallback } from 'react';
import { useNavigate, useParams } from 'react-router-dom';
import { BeakerIcon } from '../../../../pure-view/icons/BeakerIcon';
import { FeatureMakerContent } from '../../../../components/feature-maker';
import { useProjectDir } from '../../../hooks/useProjectDir';
import { useWorktreeRoute } from '../../../hooks/useWorktreeRoute';
import { fetchFeatures, updateFeature, type Feature } from '../../../../api/features';

export function FeatureDetailView() {
    const navigate = useNavigate();
    const { featureId } = useParams<{ featureId: string }>();
    const { projectDir } = useProjectDir();
    const { projectName } = useWorktreeRoute();
    const [feature, setFeature] = useState<Feature | null>(null);
    const saveTimerRef = useRef<ReturnType<typeof setTimeout> | null>(null);

    useEffect(() => {
        if (!projectName || !featureId) return;
        fetchFeatures(projectName)
            .then(features => {
                const found = features.find(f => f.id === featureId);
                setFeature(found || null);
            })
            .catch(() => setFeature(null));
    }, [projectName, featureId]);

    const debouncedSave = useCallback((updates: { title?: string; description?: string }) => {
        if (!projectName || !featureId) return;
        if (saveTimerRef.current) clearTimeout(saveTimerRef.current);
        saveTimerRef.current = setTimeout(() => {
            updateFeature(projectName, featureId, updates).catch(() => {});
        }, 500);
    }, [projectName, featureId]);

    const handleTitleChange = useCallback((title: string) => {
        setFeature(prev => prev ? { ...prev, title } : prev);
        debouncedSave({ title });
    }, [debouncedSave]);

    const handleDescriptionChange = useCallback((description: string) => {
        setFeature(prev => prev ? { ...prev, description } : prev);
        debouncedSave({ description });
    }, [debouncedSave]);

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
                initialProjectDir={projectDir || undefined}
                useRealData
                featureTitle={feature?.title ?? ''}
                featureDescription={feature?.description ?? ''}
                onFeatureTitleChange={handleTitleChange}
                onFeatureDescriptionChange={handleDescriptionChange}
            />
        </div>
    );
}
