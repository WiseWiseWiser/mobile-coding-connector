import { useState, useEffect, useCallback } from 'react';
import { useNavigate, useParams, useLocation } from 'react-router-dom';
import { BeakerIcon } from '../../../../pure-view/icons/BeakerIcon';
import { useV2Context } from '../../../V2Context';
import { ProjectPickerModal } from './ProjectPickerModal';
import { projectPath, parseWorktreeProjectName } from '../../../../route/route';
import { fetchFeatures, createFeature, deleteFeature, type Feature } from '../../../../api/features';
import './FeatureListView.css';

const statusLabels: Record<string, string> = {
    'draft': 'Draft',
    'in-progress': 'In Progress',
    'completed': 'Completed',
};

export function FeatureListView() {
    const navigate = useNavigate();
    const location = useLocation();
    const { projectName: fullProjectName } = useParams<{ projectName?: string }>();
    const { rootProjects } = useV2Context();
    const [features, setFeatures] = useState<Feature[]>([]);
    const [loading, setLoading] = useState(false);
    const [showAddForm, setShowAddForm] = useState(false);
    const [showProjectPicker, setShowProjectPicker] = useState(false);
    const [newTitle, setNewTitle] = useState('');
    const [newDescription, setNewDescription] = useState('');

    const parsed = fullProjectName ? parseWorktreeProjectName(fullProjectName) : null;
    const projectName = parsed?.projectName;

    useEffect(() => {
        if (!projectName) return;
        setLoading(true);
        fetchFeatures(projectName)
            .then(setFeatures)
            .catch(() => setFeatures([]))
            .finally(() => setLoading(false));
    }, [projectName]);

    useEffect(() => {
        const state = location.state as { showAdd?: boolean } | null;
        if (state?.showAdd && projectName) {
            setShowAddForm(true);
            navigate(location.pathname, { replace: true, state: {} });
        }
    }, [location.state, projectName, navigate, location.pathname]);

    const handleNewFeatureClick = useCallback(() => {
        if (!projectName) {
            setShowProjectPicker(true);
            return;
        }
        setShowAddForm(true);
    }, [projectName]);

    const handleProjectSelect = useCallback((selectedFullName: string) => {
        setShowProjectPicker(false);
        navigate(`${projectPath(selectedFullName)}/home/feature-maker`, { state: { showAdd: true } });
    }, [navigate]);

    const handleAdd = useCallback(async () => {
        if (!newTitle.trim() || !projectName) return;
        try {
            const feature = await createFeature(projectName, newTitle.trim(), newDescription.trim());
            setFeatures(prev => [feature, ...prev]);
            setNewTitle('');
            setNewDescription('');
            setShowAddForm(false);
        } catch {
            // TODO: show error
        }
    }, [newTitle, newDescription, projectName]);

    const handleDelete = useCallback(async (id: string) => {
        if (!projectName) return;
        try {
            await deleteFeature(projectName, id);
            setFeatures(prev => prev.filter(f => f.id !== id));
        } catch {
            // TODO: show error
        }
    }, [projectName]);

    const handleEnterFeature = useCallback((id: string) => {
        navigate(`../feature-maker/${id}`);
    }, [navigate]);

    return (
        <div className="feature-list-container">
            <div className="mcc-section-header">
                <button className="mcc-back-btn" onClick={() => navigate('../experimental')}>
                    &larr;
                </button>
                <BeakerIcon className="mcc-header-icon" />
                <h2>Feature Maker</h2>
                {projectName && (
                    <span className="feature-list-project-badge">{fullProjectName}</span>
                )}
            </div>
            <p className="mcc-section-subtitle">
                Create and manage feature requests. Each feature goes through an AI-driven implementation flow.
            </p>

            <div className="feature-list-actions">
                {!showAddForm ? (
                    <button className="feature-list-add-btn" onClick={handleNewFeatureClick}>
                        + New Feature
                    </button>
                ) : (
                    <div className="feature-list-add-form">
                        <input
                            className="feature-list-input"
                            type="text"
                            placeholder="Feature title"
                            value={newTitle}
                            onChange={e => setNewTitle(e.target.value)}
                            onKeyDown={e => e.key === 'Enter' && handleAdd()}
                            autoFocus
                        />
                        <textarea
                            className="feature-list-textarea"
                            placeholder="Description (optional)"
                            value={newDescription}
                            onChange={e => setNewDescription(e.target.value)}
                            rows={3}
                        />
                        <div className="feature-list-form-actions">
                            <button className="feature-list-save-btn" onClick={handleAdd} disabled={!newTitle.trim()}>
                                Create
                            </button>
                            <button className="feature-list-cancel-btn" onClick={() => { setShowAddForm(false); setNewTitle(''); setNewDescription(''); }}>
                                Cancel
                            </button>
                        </div>
                    </div>
                )}
            </div>

            {loading ? (
                <div className="feature-list-empty">
                    <p>Loading...</p>
                </div>
            ) : features.length === 0 && !showAddForm ? (
                <div className="feature-list-empty">
                    <BeakerIcon className="feature-list-empty-icon" />
                    <p>No features yet</p>
                    <p className="feature-list-empty-hint">Click "New Feature" to get started</p>
                </div>
            ) : (
                <div className="feature-list-items">
                    {features.map(feature => (
                        <div
                            key={feature.id}
                            className="feature-list-card"
                            onClick={() => handleEnterFeature(feature.id)}
                        >
                            <div className="feature-list-card-header">
                                <h3 className="feature-list-card-title">{feature.title}</h3>
                                <span className={`feature-list-card-status feature-list-status-${feature.status}`}>
                                    {statusLabels[feature.status] || feature.status}
                                </span>
                            </div>
                            {feature.description && (
                                <p className="feature-list-card-desc">{feature.description}</p>
                            )}
                            <div className="feature-list-card-footer">
                                <span className="feature-list-card-date">
                                    {new Date(feature.created_at).toLocaleDateString()}
                                </span>
                                <button
                                    className="feature-list-card-delete"
                                    onClick={e => { e.stopPropagation(); handleDelete(feature.id); }}
                                >
                                    Delete
                                </button>
                            </div>
                        </div>
                    ))}
                </div>
            )}

            {showProjectPicker && (
                <ProjectPickerModal
                    projects={rootProjects}
                    onSelect={handleProjectSelect}
                    onClose={() => setShowProjectPicker(false)}
                />
            )}
        </div>
    );
}
