import { useState, useEffect } from 'react';
import { fetchFileContent, saveFileContent } from '../../../api/files';
import './FileActionSheet.css';

interface FileEditorProps {
    filePath: string;
    basePath: string;
    onClose: () => void;
    onSave: () => void;
}

export function FileEditor({ filePath, basePath, onClose, onSave }: FileEditorProps) {
    const [content, setContent] = useState('');
    const [originalContent, setOriginalContent] = useState('');
    const [loading, setLoading] = useState(true);
    const [saving, setSaving] = useState(false);
    const [error, setError] = useState('');

    useEffect(() => {
        loadFile();
    }, [filePath]);

    const loadFile = async () => {
        setLoading(true);
        setError('');
        try {
            const fullPath = basePath + (filePath ? '/' + filePath : '');
            const data = await fetchFileContent(fullPath);
            setContent(data);
            setOriginalContent(data);
        } catch (e) {
            setError(e instanceof Error ? e.message : 'Failed to load file');
        } finally {
            setLoading(false);
        }
    };

    const handleSave = async () => {
        setSaving(true);
        setError('');
        try {
            const fullPath = basePath + (filePath ? '/' + filePath : '');
            await saveFileContent(fullPath, content);
            setOriginalContent(content);
            onSave();
        } catch (e) {
            setError(e instanceof Error ? e.message : 'Failed to save file');
        } finally {
            setSaving(false);
        }
    };

    const handleCancel = () => {
        if (content !== originalContent) {
            if (confirm('You have unsaved changes. Are you sure you want to discard them?')) {
                onClose();
            }
        } else {
            onClose();
        }
    };

    const hasChanges = content !== originalContent;

    return (
        <div className="file-editor-overlay" onClick={handleCancel}>
            <div className="file-editor" onClick={(e) => e.stopPropagation()}>
                <div className="file-editor-header">
                    <span className="file-editor-title">
                        {filePath.split('/').pop()}
                        {hasChanges && <span style={{ color: '#f59e0b', marginLeft: 8 }}>●</span>}
                    </span>
                    <button className="file-editor-close" onClick={handleCancel}>×</button>
                </div>
                
                <div className="file-editor-content">
                    {loading ? (
                        <div style={{ textAlign: 'center', padding: 40, color: '#94a3b8' }}>Loading...</div>
                    ) : error ? (
                        <div style={{ textAlign: 'center', padding: 40, color: '#ef4444' }}>{error}</div>
                    ) : (
                        <textarea
                            className="file-editor-textarea"
                            value={content}
                            onChange={(e) => setContent(e.target.value)}
                            placeholder="Start typing..."
                            spellCheck={false}
                        />
                    )}
                </div>

                <div className="file-editor-actions">
                    <button 
                        className="file-editor-btn save" 
                        onClick={handleSave}
                        disabled={saving || loading || !hasChanges}
                    >
                        {saving ? 'Saving...' : 'Save'}
                    </button>
                    <button 
                        className="file-editor-btn cancel" 
                        onClick={handleCancel}
                        disabled={saving}
                    >
                        Cancel
                    </button>
                </div>
            </div>
        </div>
    );
}
