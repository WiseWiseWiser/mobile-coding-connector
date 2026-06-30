import { useCallback, useEffect, useRef, useState } from 'react';
import { useNavigate } from 'react-router-dom';
import {
    deleteFileTransfer,
    fileTransferDownloadUrl,
    getScratch,
    listFileTransfer,
    saveScratch,
    uploadFileTransfer,
} from '../../../api/fileTransfer';
import type { FileTransferEntry } from '../../../api/fileTransfer';
import { PageView } from '../../../pure-view/PageView';
import { UploadIcon } from '../../../pure-view/icons/UploadIcon';
import './FileTransferView.css';

function formatFileSize(bytes: number): string {
    if (bytes === 0) return '0 B';
    const units = ['B', 'KB', 'MB', 'GB'];
    const i = Math.floor(Math.log(bytes) / Math.log(1024));
    const size = bytes / Math.pow(1024, i);
    return `${size.toFixed(i > 0 ? 1 : 0)} ${units[i]}`;
}

function formatRelativeDate(iso: string): string {
    const date = new Date(iso);
    if (Number.isNaN(date.getTime())) return '';
    const now = Date.now();
    const diffMs = now - date.getTime();
    const diffMins = Math.floor(diffMs / 60000);
    if (diffMins < 1) return 'just now';
    if (diffMins < 60) return `${diffMins}m ago`;
    const diffHours = Math.floor(diffMins / 60);
    if (diffHours < 24) return `${diffHours}h ago`;
    const diffDays = Math.floor(diffHours / 24);
    if (diffDays < 7) return `${diffDays}d ago`;
    return date.toLocaleDateString();
}

export function FileTransferView() {
    const navigate = useNavigate();
    const fileInputRef = useRef<HTMLInputElement>(null);
    const [files, setFiles] = useState<FileTransferEntry[]>([]);
    const [loading, setLoading] = useState(true);
    const [uploading, setUploading] = useState(false);
    const [error, setError] = useState<string | null>(null);
    const [dragOver, setDragOver] = useState(false);
    const [scratchContent, setScratchContent] = useState('');
    const [scratchLoading, setScratchLoading] = useState(true);
    const [scratchSaving, setScratchSaving] = useState(false);

    const loadScratch = useCallback(async () => {
        setScratchLoading(true);
        try {
            const scratch = await getScratch();
            setScratchContent(scratch.content);
        } catch (err) {
            setError(err instanceof Error ? err.message : String(err));
        } finally {
            setScratchLoading(false);
        }
    }, []);

    const refresh = useCallback(async () => {
        setError(null);
        try {
            const list = await listFileTransfer();
            setFiles(list);
        } catch (err) {
            setError(err instanceof Error ? err.message : String(err));
        } finally {
            setLoading(false);
        }
    }, []);

    useEffect(() => {
        void refresh();
        void loadScratch();
    }, [refresh, loadScratch]);

    const handleScratchSave = async () => {
        setScratchSaving(true);
        setError(null);
        try {
            const saved = await saveScratch(scratchContent);
            setScratchContent(saved.content);
        } catch (err) {
            setError(err instanceof Error ? err.message : String(err));
        } finally {
            setScratchSaving(false);
        }
    };

    const handleScratchCopy = async () => {
        setError(null);
        try {
            await navigator.clipboard.writeText(scratchContent);
        } catch (err) {
            setError(err instanceof Error ? err.message : String(err));
        }
    };

    const handleUploadFile = async (file: File) => {
        setUploading(true);
        setError(null);
        try {
            await uploadFileTransfer(file);
            await refresh();
        } catch (err) {
            setError(err instanceof Error ? err.message : String(err));
        } finally {
            setUploading(false);
            if (fileInputRef.current) {
                fileInputRef.current.value = '';
            }
        }
    };

    const handleFileInputChange = (e: React.ChangeEvent<HTMLInputElement>) => {
        const file = e.target.files?.[0];
        if (file) {
            void handleUploadFile(file);
        }
    };

    const handleChooseFile = () => {
        fileInputRef.current?.click();
    };

    const handleDrop = (e: React.DragEvent) => {
        e.preventDefault();
        setDragOver(false);
        const file = e.dataTransfer.files?.[0];
        if (file) {
            void handleUploadFile(file);
        }
    };

    const handleDownload = (name: string) => {
        const a = document.createElement('a');
        a.href = fileTransferDownloadUrl(name);
        a.download = name;
        a.rel = 'noopener';
        document.body.appendChild(a);
        a.click();
        document.body.removeChild(a);
    };

    const handleRemove = async (name: string) => {
        if (!window.confirm(`Remove "${name}" from file transfer?`)) {
            return;
        }
        setError(null);
        try {
            await deleteFileTransfer(name);
            await refresh();
        } catch (err) {
            setError(err instanceof Error ? err.message : String(err));
        }
    };

    return (
        <PageView>
            <div className="mcc-section-header">
                <button className="mcc-back-btn" onClick={() => navigate('..')}>&larr;</button>
                <h2>File Transfer</h2>
            </div>

            <div className="file-transfer-body">
                <section
                    className="file-transfer-scratch"
                    data-testid="file-transfer-scratch"
                >
                    <h3 className="file-transfer-scratch-heading">Quick Transfer</h3>
                    <textarea
                        className="file-transfer-scratch-input"
                        data-testid="file-transfer-scratch-input"
                        value={scratchContent}
                        onChange={(e) => setScratchContent(e.target.value)}
                        placeholder="Paste text here to share across devices…"
                        disabled={scratchLoading || scratchSaving}
                        rows={4}
                    />
                    <div className="file-transfer-scratch-actions">
                        <button
                            type="button"
                            className="file-transfer-scratch-btn"
                            data-testid="file-transfer-scratch-save"
                            onClick={() => void handleScratchSave()}
                            disabled={scratchLoading || scratchSaving}
                        >
                            {scratchSaving ? 'Saving…' : 'Save'}
                        </button>
                        <button
                            type="button"
                            className="file-transfer-scratch-btn file-transfer-scratch-btn--secondary"
                            data-testid="file-transfer-scratch-copy"
                            onClick={() => void handleScratchCopy()}
                            disabled={scratchLoading}
                        >
                            Copy
                        </button>
                    </div>
                </section>

                <div
                    className={`file-transfer-upload${dragOver ? ' file-transfer-upload--drag-over' : ''}`}
                    data-testid="file-transfer-upload"
                    onDragOver={(e) => { e.preventDefault(); setDragOver(true); }}
                    onDragLeave={() => setDragOver(false)}
                    onDrop={handleDrop}
                >
                    <input
                        ref={fileInputRef}
                        type="file"
                        className="file-transfer-file-input"
                        onChange={handleFileInputChange}
                        disabled={uploading}
                    />
                    <button
                        type="button"
                        className="file-transfer-upload-btn"
                        onClick={handleChooseFile}
                        disabled={uploading}
                    >
                        <UploadIcon />
                        <span>{uploading ? 'Uploading…' : 'Upload'}</span>
                    </button>
                    <p className="file-transfer-upload-hint">or drag and drop a file here</p>
                </div>

                {error && <div className="file-transfer-error">{error}</div>}
                {uploading && <div className="file-transfer-progress">Uploading…</div>}

                {loading ? (
                    <div className="file-transfer-loading">Loading files…</div>
                ) : files.length === 0 ? (
                    <div className="file-transfer-empty">
                        No files yet — upload a file to get started
                    </div>
                ) : (
                    <ul className="file-transfer-list">
                        {files.map((file) => (
                            <li
                                key={file.name}
                                className="file-transfer-row"
                                data-testid="file-transfer-row"
                            >
                                <div className="file-transfer-row-main">
                                    <span className="file-transfer-row-name">{file.name}</span>
                                    <span className="file-transfer-row-meta">
                                        {formatFileSize(file.size)} · {formatRelativeDate(file.uploaded_at)}
                                    </span>
                                </div>
                                <div className="file-transfer-row-actions">
                                    <button
                                        type="button"
                                        className="file-transfer-action-btn"
                                        onClick={() => handleDownload(file.name)}
                                    >
                                        Download
                                    </button>
                                    <button
                                        type="button"
                                        className="file-transfer-action-btn file-transfer-action-btn--remove"
                                        onClick={() => void handleRemove(file.name)}
                                    >
                                        Remove
                                    </button>
                                </div>
                            </li>
                        ))}
                    </ul>
                )}
            </div>
        </PageView>
    );
}