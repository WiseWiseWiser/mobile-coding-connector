import { useState, useEffect } from 'react';
import { fetchFileContent } from '../../../api/checkpoints';
import './FilesView.css';

export interface FileContentViewProps {
    projectDir: string;
    filePath: string;
    onBack: () => void;
}

export function FileContentView({ projectDir, filePath, onBack }: FileContentViewProps) {
    const [content, setContent] = useState<string | null>(null);
    const [loading, setLoading] = useState(true);
    const [error, setError] = useState<string | null>(null);
    const [wordWrap, setWordWrap] = useState(true);

    useEffect(() => {
        setLoading(true);
        setError(null);
        fetchFileContent(projectDir, filePath)
            .then(data => { setContent(data); setLoading(false); })
            .catch(err => { setError(err instanceof Error ? err.message : 'Failed to load file'); setLoading(false); });
    }, [projectDir, filePath]);

    const fileName = filePath.includes('/') ? filePath.substring(filePath.lastIndexOf('/') + 1) : filePath;

    return (
        <div className="mcc-files">
            <div className="mcc-section-header">
                <button className="mcc-back-btn" onClick={onBack}>&larr;</button>
                <h2 className="mcc-file-viewer-title">{fileName}</h2>
            </div>
            <div className="mcc-file-viewer-toolbar">
                <span className="mcc-file-viewer-path-inline">{filePath}</span>
                <label className="mcc-file-viewer-wrap-toggle">
                    <input type="checkbox" checked={wordWrap} onChange={e => setWordWrap(e.target.checked)} />
                    <span>Wrap</span>
                </label>
            </div>
            {loading ? (
                <div className="mcc-files-empty">Loading file...</div>
            ) : error ? (
                <div className="mcc-checkpoint-error">{error}</div>
            ) : (
                <div className="mcc-file-viewer-content">
                    <pre className={`mcc-file-viewer-code${wordWrap ? ' mcc-file-viewer-code-wrap' : ''}`}>{content}</pre>
                </div>
            )}
        </div>
    );
}
