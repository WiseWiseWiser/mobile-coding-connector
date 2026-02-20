import { useState } from 'react';
import { useNavigate } from 'react-router-dom';
import { addProject, updateProject } from '../../../api/projects';
import { useV2Context } from '../../V2Context';
import { ServerFileBrowser } from './ServerFileBrowser';
import './AddFromFilesystemView.css';

export function AddFromFilesystemView() {
    const navigate = useNavigate();
    const { fetchProjects, projectsList } = useV2Context();

    const [selectedDir, setSelectedDir] = useState('');
    const [name, setName] = useState('');
    const [parentId, setParentId] = useState('');
    const [adding, setAdding] = useState(false);
    const [error, setError] = useState<string | null>(null);
    const [success, setSuccess] = useState<string | null>(null);

    // Get root projects (projects without parent) for the dropdown
    const rootProjects = projectsList.filter(p => !p.parent_id);

    const handleDirectoryChange = (dir: string) => {
        setSelectedDir(dir);
        setError(null);
        setSuccess(null);
    };

    const handleAdd = async () => {
        if (!selectedDir) {
            setError('Please select a directory');
            return;
        }
        setAdding(true);
        setError(null);
        setSuccess(null);
        try {
            const result = await addProject({ dir: selectedDir, name: name.trim() || undefined });
            
            // If a parent project is selected, update the new project to be a subproject
            if (parentId) {
                await updateProject(result.id, { parent_id: parentId });
            }
            
            setSuccess(`Added workspace: ${result.name} (${result.dir})`);
            fetchProjects();
        } catch (err) {
            setError(err instanceof Error ? err.message : String(err));
        }
        setAdding(false);
    };

    return (
        <div className="add-fs-view">
            <div className="mcc-section-header">
                <button className="mcc-back-btn" onClick={() => navigate('..')}>&larr;</button>
                <h2>Add From Filesystem</h2>
            </div>

            <div className="add-fs-form">
                {/* Name (optional) */}
                <div className="add-fs-section">
                    <label className="add-fs-label">Workspace Name (optional)</label>
                    <input
                        type="text"
                        className="add-fs-input"
                        placeholder="Leave empty to use directory name"
                        value={name}
                        onChange={e => setName(e.target.value)}
                    />
                </div>

                {/* Parent Project (optional) */}
                {rootProjects.length > 0 && (
                    <div className="add-fs-section">
                        <label className="add-fs-label">Parent Project (optional)</label>
                        <select
                            className="add-fs-input"
                            value={parentId}
                            onChange={e => setParentId(e.target.value)}
                        >
                            <option value="">-- No parent (root project) --</option>
                            {rootProjects.map(p => (
                                <option key={p.id} value={p.id}>
                                    {p.name}
                                </option>
                            ))}
                        </select>
                        <div className="add-fs-hint">
                            Select a parent project to create this as a sub-project
                        </div>
                    </div>
                )}

                {/* Selected directory display */}
                <div className="add-fs-section">
                    <label className="add-fs-label">Selected Directory</label>
                    <div className="add-fs-selected-dir">
                        {selectedDir || 'Browse below to select a directory'}
                    </div>
                </div>

                {/* Server File Browser */}
                <div className="add-fs-section add-fs-browser-section">
                    <label className="add-fs-label">Browse Server Filesystem</label>
                    <ServerFileBrowser
                        selectMode="dir"
                        onDirectoryChange={handleDirectoryChange}
                    />
                </div>

                {/* Error */}
                {error && <div className="add-fs-error">{error}</div>}

                {/* Success */}
                {success && <div className="add-fs-success">{success}</div>}

                {/* Add Button */}
                <button
                    className="add-fs-submit-btn"
                    onClick={handleAdd}
                    disabled={adding || !selectedDir}
                >
                    {adding ? 'Adding...' : 'Add Workspace'}
                </button>
            </div>
        </div>
    );
}
