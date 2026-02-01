import { DiffEditor } from '@monaco-editor/react';
import type { DiffFile } from './types';
import { parseDiffContent, getFileLanguage } from './utils';

interface DiffViewerProps {
    selectedFile: DiffFile | null;
}

export function DiffViewer({ selectedFile }: DiffViewerProps) {
    if (!selectedFile) {
        return (
            <div style={{ 
                flex: 1, 
                display: 'flex', 
                alignItems: 'center', 
                justifyContent: 'center',
                color: '#9ca3af',
                fontSize: '14px',
            }}>
                Select a file to view diff
            </div>
        );
    }

    const { original, modified } = parseDiffContent(selectedFile.diff);

    return (
        <>
            <div style={{ 
                padding: '8px 16px', 
                borderBottom: '1px solid #e5e5e5',
                backgroundColor: '#fff',
                fontSize: '13px',
                fontFamily: 'monospace',
            }}>
                {selectedFile.path}
                {selectedFile.isStaged && (
                    <span style={{ 
                        marginLeft: '8px', 
                        padding: '2px 6px', 
                        backgroundColor: '#dcfce7',
                        color: '#166534',
                        borderRadius: '4px',
                        fontSize: '11px',
                    }}>
                        staged
                    </span>
                )}
            </div>
            <div style={{ flex: 1 }}>
                <DiffEditor
                    original={original}
                    modified={modified}
                    language={getFileLanguage(selectedFile.path)}
                    theme="vs-dark"
                    options={{
                        readOnly: true,
                        renderSideBySide: true,
                        minimap: { enabled: false },
                        fontSize: 13,
                        lineNumbers: 'on',
                        scrollBeyondLastLine: false,
                    }}
                />
            </div>
        </>
    );
}
