import { useState, useRef } from 'react';
import { useNavigate } from 'react-router-dom';
import { checkServerFile, uploadFile } from '../../../api/fileupload';
import type { ServerFileInfo } from '../../../api/fileupload';
import { ServerFileBrowser } from './ServerFileBrowser';
import { TransferProgress } from './TransferProgress';
import type { TransferProgressData } from './TransferProgress';
import './UploadFileView.css';

function formatFileSize(bytes: number): string {
    if (bytes === 0) return '0 B';
    const units = ['B', 'KB', 'MB', 'GB'];
    const i = Math.floor(Math.log(bytes) / Math.log(1024));
    const size = bytes / Math.pow(1024, i);
    return `${size.toFixed(i > 0 ? 1 : 0)} ${units[i]}`;
}

export function UploadFileView() {
    const navigate = useNavigate();
    const fileInputRef = useRef<HTMLInputElement>(null);

    const [selectedFile, setSelectedFile] = useState<File | null>(null);
    const [serverPath, setServerPath] = useState('');
    const [serverFileInfo, setServerFileInfo] = useState<ServerFileInfo | null>(null);
    const [checking, setChecking] = useState(false);
    const [uploading, setUploading] = useState(false);
    const [uploadProgress, setUploadProgress] = useState<TransferProgressData | null>(null);
    const [uploadSuccess, setUploadSuccess] = useState(false);
    const [error, setError] = useState<string | null>(null);
    const [showBrowser, setShowBrowser] = useState(false);

    const handleFileChange = (e: React.ChangeEvent<HTMLInputElement>) => {
        const file = e.target.files?.[0] ?? null;
        setSelectedFile(file);
        setUploadSuccess(false);
        setError(null);
        setServerFileInfo(null);
    };

    const handleChooseFile = () => {
        fileInputRef.current?.click();
    };

    const handleCheckPath = async () => {
        if (!serverPath.trim()) {
            setError('Please enter a server path');
            return;
        }
        setChecking(true);
        setError(null);
        setServerFileInfo(null);
        try {
            const info = await checkServerFile(serverPath.trim());
            setServerFileInfo(info);
        } catch (err) {
            setError(err instanceof Error ? err.message : String(err));
        }
        setChecking(false);
    };

    const handleUpload = async () => {
        if (!selectedFile) {
            setError('Please select a file');
            return;
        }
        if (!serverPath.trim()) {
            setError('Please enter a server path');
            return;
        }
        setUploading(true);
        setError(null);
        setUploadSuccess(false);
        setUploadProgress(null);
        try {
            await uploadFile(selectedFile, serverPath.trim(), setUploadProgress);
            setUploadSuccess(true);
            // Re-check file info after upload
            const info = await checkServerFile(serverPath.trim());
            setServerFileInfo(info);
        } catch (err) {
            setError(err instanceof Error ? err.message : String(err));
        }
        setUploading(false);
    };

    const handleBrowserSelect = (path: string | null) => {
        if (!path) return;
        setServerPath(path);
        setServerFileInfo(null);
        setUploadSuccess(false);
        setShowBrowser(false);
    };

    const handleBrowserDirChange = (dirPath: string) => {
        // When the user navigates to a directory and the selected file is known,
        // auto-compose path: dir + filename
        if (selectedFile) {
            setServerPath(dirPath.replace(/\/$/, '') + '/' + selectedFile.name);
            setServerFileInfo(null);
            setUploadSuccess(false);
        }
    };

    const showOverwriteWarning = serverFileInfo?.exists && !serverFileInfo.is_dir && !uploadSuccess;

    return (
        <div className="upload-view">
            <div className="mcc-section-header">
                <button className="mcc-back-btn" onClick={() => navigate('..')}>&larr;</button>
                <h2>Upload File</h2>
            </div>

            <div className="upload-form">
                {/* File Selection */}
                <div className="upload-section">
                    <label className="upload-label">Local File</label>
                    <input
                        ref={fileInputRef}
                        type="file"
                        onChange={handleFileChange}
                        style={{ display: 'none' }}
                    />
                    <button className="upload-choose-btn" onClick={handleChooseFile}>
                        {selectedFile ? 'Change File' : 'Choose File'}
                    </button>
                    {selectedFile && (
                        <div className="upload-file-info">
                            <div className="upload-file-info-row">
                                <span className="upload-file-info-label">Name:</span>
                                <span className="upload-file-info-value">{selectedFile.name}</span>
                            </div>
                            <div className="upload-file-info-row">
                                <span className="upload-file-info-label">Size:</span>
                                <span className="upload-file-info-value">{formatFileSize(selectedFile.size)}</span>
                            </div>
                            <div className="upload-file-info-row">
                                <span className="upload-file-info-label">Type:</span>
                                <span className="upload-file-info-value">{selectedFile.type || 'unknown'}</span>
                            </div>
                            <div className="upload-file-info-row">
                                <span className="upload-file-info-label">Modified:</span>
                                <span className="upload-file-info-value">{new Date(selectedFile.lastModified).toLocaleString()}</span>
                            </div>
                        </div>
                    )}
                </div>

                {/* Server Path */}
                <div className="upload-section">
                    <label className="upload-label">Server Destination Path</label>
                    <div className="upload-path-row">
                        <input
                            type="text"
                            className="upload-path-input"
                            placeholder="/path/to/destination/file.txt"
                            value={serverPath}
                            onChange={e => { setServerPath(e.target.value); setServerFileInfo(null); setUploadSuccess(false); }}
                        />
                        <button
                            className="upload-check-btn"
                            onClick={() => setShowBrowser(!showBrowser)}
                        >
                            Browse
                        </button>
                        <button
                            className="upload-check-btn"
                            onClick={handleCheckPath}
                            disabled={checking || !serverPath.trim()}
                        >
                            {checking ? 'Checking...' : 'Check'}
                        </button>
                    </div>
                </div>

                {/* Server File Browser (toggle) */}
                {showBrowser && (
                    <div className="upload-section">
                        <ServerFileBrowser
                            selectMode="file_or_dir"
                            onSelect={handleBrowserSelect}
                            onDirectoryChange={handleBrowserDirChange}
                        />
                    </div>
                )}

                {/* Server File Info */}
                {serverFileInfo && (
                    <div className="upload-section">
                        <label className="upload-label">Server File Status</label>
                        {serverFileInfo.exists ? (
                            <div className="upload-server-info">
                                <div className="upload-file-info-row">
                                    <span className="upload-file-info-label">Status:</span>
                                    <span className="upload-file-info-value upload-file-exists">
                                        {serverFileInfo.is_dir ? 'Directory exists' : 'File exists'}
                                    </span>
                                </div>
                                <div className="upload-file-info-row">
                                    <span className="upload-file-info-label">Path:</span>
                                    <code className="upload-file-info-code">{serverFileInfo.path}</code>
                                </div>
                                <div className="upload-file-info-row">
                                    <span className="upload-file-info-label">Size:</span>
                                    <span className="upload-file-info-value">{formatFileSize(serverFileInfo.size)}</span>
                                </div>
                                {serverFileInfo.mod_time && (
                                    <div className="upload-file-info-row">
                                        <span className="upload-file-info-label">Modified:</span>
                                        <span className="upload-file-info-value">{new Date(serverFileInfo.mod_time).toLocaleString()}</span>
                                    </div>
                                )}
                                {serverFileInfo.file_mode && (
                                    <div className="upload-file-info-row">
                                        <span className="upload-file-info-label">Permissions:</span>
                                        <code className="upload-file-info-code">{serverFileInfo.file_mode}</code>
                                    </div>
                                )}
                            </div>
                        ) : (
                            <div className="upload-server-info">
                                <div className="upload-file-info-row">
                                    <span className="upload-file-info-label">Status:</span>
                                    <span className="upload-file-info-value upload-file-not-exists">File does not exist (will be created)</span>
                                </div>
                                <div className="upload-file-info-row">
                                    <span className="upload-file-info-label">Path:</span>
                                    <code className="upload-file-info-code">{serverFileInfo.path}</code>
                                </div>
                            </div>
                        )}
                    </div>
                )}

                {/* Overwrite Warning */}
                {showOverwriteWarning && (
                    <div className="upload-warning">
                        This file already exists on the server and will be overwritten.
                    </div>
                )}

                {/* Error */}
                {error && <div className="upload-error">{error}</div>}

                {/* Upload Progress */}
                {uploading && <TransferProgress progress={uploadProgress} label="Upload" />}

                {/* Success */}
                {uploadSuccess && <div className="upload-success">File uploaded successfully!</div>}

                {/* Upload Button */}
                <button
                    className="upload-submit-btn"
                    onClick={handleUpload}
                    disabled={uploading || !selectedFile || !serverPath.trim()}
                >
                    {uploading ? 'Uploading...' : showOverwriteWarning ? 'Overwrite & Upload' : 'Upload'}
                </button>
            </div>
        </div>
    );
}
