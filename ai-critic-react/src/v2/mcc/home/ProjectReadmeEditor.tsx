import { useState, useEffect, useCallback } from 'react';
import { fetchReadme, updateReadme } from '../../../api/projects';

interface ProjectReadmeEditorProps {
    projectId: string;
}

export function ProjectReadmeEditor({ projectId }: ProjectReadmeEditorProps) {
    const [readme, setReadme] = useState('');
    const [isEditing, setIsEditing] = useState(false);
    const [isLoading, setIsLoading] = useState(true);
    const [isSaving, setIsSaving] = useState(false);
    const [error, setError] = useState('');

    const loadReadme = useCallback(async () => {
        setIsLoading(true);
        setError('');
        try {
            const content = await fetchReadme(projectId);
            setReadme(content);
        } catch (err) {
            setError('Failed to load README');
            console.error('Error loading README:', err);
        } finally {
            setIsLoading(false);
        }
    }, [projectId]);

    useEffect(() => {
        loadReadme();
    }, [loadReadme]);

    const handleSave = async () => {
        setIsSaving(true);
        setError('');
        try {
            await updateReadme(projectId, readme);
            setIsEditing(false);
        } catch (err) {
            setError('Failed to save README');
            console.error('Error saving README:', err);
        } finally {
            setIsSaving(false);
        }
    };

    const handleCancel = () => {
        setIsEditing(false);
        loadReadme();
    };

    const renderMarkdown = (content: string) => {
        if (!content.trim()) {
            return (
                <div style={{ 
                    padding: '20px', 
                    textAlign: 'center', 
                    color: '#64748b',
                    fontSize: '14px',
                    fontStyle: 'italic'
                }}>
                    No README content yet. Click "Edit" to add documentation for this project.
                </div>
            );
        }

        const lines = content.split('\n');
        const elements: React.ReactElement[] = [];
        let inCodeBlock = false;
        let codeBlockContent: string[] = [];

        lines.forEach((line, idx) => {
            const trimmedLine = line.trim();

            if (trimmedLine.startsWith('```')) {
                if (inCodeBlock) {
                    elements.push(
                        <pre key={`code-${idx}`} style={{
                            background: '#0f172a',
                            padding: '12px',
                            borderRadius: '6px',
                            overflow: 'auto',
                            fontSize: '13px',
                            lineHeight: '1.5',
                            margin: '8px 0',
                            fontFamily: 'monospace',
                            color: '#e2e8f0',
                            border: '1px solid #334155'
                        }}>
                            <code>{codeBlockContent.join('\n')}</code>
                        </pre>
                    );
                    inCodeBlock = false;
                    codeBlockContent = [];
                } else {
                    inCodeBlock = true;
                }
                return;
            }

            if (inCodeBlock) {
                codeBlockContent.push(line);
                return;
            }

            if (trimmedLine.startsWith('# ')) {
                elements.push(
                    <h1 key={idx} style={{
                        fontSize: '20px',
                        fontWeight: 600,
                        color: '#e2e8f0',
                        margin: '16px 0 12px 0',
                        paddingBottom: '8px',
                        borderBottom: '1px solid #334155'
                    }}>
                        {trimmedLine.slice(2)}
                    </h1>
                );
            } else if (trimmedLine.startsWith('## ')) {
                elements.push(
                    <h2 key={idx} style={{
                        fontSize: '17px',
                        fontWeight: 600,
                        color: '#e2e8f0',
                        margin: '14px 0 10px 0'
                    }}>
                        {trimmedLine.slice(3)}
                    </h2>
                );
            } else if (trimmedLine.startsWith('### ')) {
                elements.push(
                    <h3 key={idx} style={{
                        fontSize: '15px',
                        fontWeight: 600,
                        color: '#cbd5e1',
                        margin: '12px 0 8px 0'
                    }}>
                        {trimmedLine.slice(4)}
                    </h3>
                );
            } else if (trimmedLine.startsWith('- ') || trimmedLine.startsWith('* ')) {
                elements.push(
                    <div key={idx} style={{
                        display: 'flex',
                        alignItems: 'flex-start',
                        margin: '4px 0',
                        paddingLeft: '16px'
                    }}>
                        <span style={{ 
                            color: '#60a5fa', 
                            marginRight: '8px',
                            fontSize: '14px',
                            lineHeight: '1.5'
                        }}>•</span>
                        <span style={{ 
                            color: '#cbd5e1',
                            fontSize: '14px',
                            lineHeight: '1.5',
                            flex: 1
                        }}>{trimmedLine.slice(2)}</span>
                    </div>
                );
            } else if (trimmedLine.match(/^\d+\.\s/)) {
                const match = trimmedLine.match(/^(\d+)\.\s/);
                if (match) {
                    elements.push(
                        <div key={idx} style={{
                            display: 'flex',
                            alignItems: 'flex-start',
                            margin: '4px 0',
                            paddingLeft: '16px'
                        }}>
                            <span style={{ 
                                color: '#60a5fa', 
                                marginRight: '8px',
                                fontSize: '13px',
                                fontWeight: 500,
                                minWidth: '20px'
                            }}>{match[1]}.</span>
                            <span style={{ 
                                color: '#cbd5e1',
                                fontSize: '14px',
                                lineHeight: '1.5',
                                flex: 1
                            }}>{trimmedLine.slice(match[0].length)}</span>
                        </div>
                    );
                }
            } else if (trimmedLine.startsWith('> ')) {
                elements.push(
                    <blockquote key={idx} style={{
                        borderLeft: '3px solid #60a5fa',
                        paddingLeft: '12px',
                        margin: '8px 0',
                        color: '#94a3b8',
                        fontStyle: 'italic',
                        fontSize: '14px',
                        lineHeight: '1.5'
                    }}>
                        {trimmedLine.slice(2)}
                    </blockquote>
                );
            } else if (trimmedLine === '---' || trimmedLine === '***' || trimmedLine === '___') {
                elements.push(
                    <hr key={idx} style={{
                        border: 'none',
                        borderTop: '1px solid #334155',
                        margin: '16px 0'
                    }} />
                );
            } else if (trimmedLine === '') {
                elements.push(<div key={idx} style={{ height: '8px' }} />);
            } else {
                elements.push(
                    <p key={idx} style={{
                        color: '#cbd5e1',
                        fontSize: '14px',
                        lineHeight: '1.6',
                        margin: '6px 0'
                    }}>
                        {line}
                    </p>
                );
            }
        });

        return (
            <div style={{ padding: '12px' }}>
                {elements}
            </div>
        );
    };

    if (isLoading) {
        return (
            <div style={{ padding: '16px' }}>
                <div style={{ 
                    fontSize: '15px', 
                    fontWeight: 600, 
                    color: '#e2e8f0', 
                    marginBottom: 12 
                }}>
                    Project README
                </div>
                <div style={{ 
                    padding: '20px', 
                    textAlign: 'center',
                    color: '#64748b',
                    fontSize: '14px'
                }}>
                    Loading...
                </div>
            </div>
        );
    }

    return (
        <div style={{ padding: '16px' }}>
            <div style={{ 
                display: 'flex', 
                justifyContent: 'space-between', 
                alignItems: 'center',
                marginBottom: 12 
            }}>
                <div style={{ 
                    fontSize: '15px', 
                    fontWeight: 600, 
                    color: '#e2e8f0' 
                }}>
                    Project README
                </div>
                {!isEditing && (
                    <button
                        onClick={() => setIsEditing(true)}
                        style={{
                            padding: '6px 12px',
                            background: '#3b82f6',
                            color: '#fff',
                            border: 'none',
                            borderRadius: 6,
                            fontSize: '13px',
                            cursor: 'pointer'
                        }}
                    >
                        Edit
                    </button>
                )}
            </div>

            {error && (
                <div style={{
                    padding: '10px 14px',
                    background: 'rgba(239, 68, 68, 0.1)',
                    border: '1px solid rgba(239, 68, 68, 0.3)',
                    borderRadius: 8,
                    color: '#fca5a5',
                    fontSize: '13px',
                    marginBottom: 12
                }}>
                    {error}
                </div>
            )}

            {isEditing ? (
                <div>
                    <textarea
                        value={readme}
                        onChange={(e) => setReadme(e.target.value)}
                        placeholder="# Project README

Write your project documentation here...

## Overview
- Key features
- Setup instructions
- Important notes"
                        style={{
                            width: '100%',
                            minHeight: '300px',
                            padding: '12px',
                            background: '#0f172a',
                            border: '1px solid #334155',
                            borderRadius: 8,
                            color: '#e2e8f0',
                            fontSize: '14px',
                            lineHeight: '1.6',
                            fontFamily: 'monospace',
                            resize: 'vertical'
                        }}
                    />
                    <div style={{ 
                        display: 'flex', 
                        gap: 10, 
                        marginTop: 12,
                        justifyContent: 'flex-end'
                    }}>
                        <button
                            onClick={handleSave}
                            disabled={isSaving}
                            style={{
                                padding: '8px 16px',
                                background: '#3b82f6',
                                color: '#fff',
                                border: 'none',
                                borderRadius: 6,
                                fontSize: '14px',
                                fontWeight: 600,
                                cursor: isSaving ? 'not-allowed' : 'pointer',
                                opacity: isSaving ? 0.7 : 1
                            }}
                        >
                            {isSaving ? 'Saving...' : 'Save'}
                        </button>
                        <button
                            onClick={handleCancel}
                            disabled={isSaving}
                            style={{
                                padding: '8px 16px',
                                background: '#1e293b',
                                color: '#94a3b8',
                                border: '1px solid #334155',
                                borderRadius: 6,
                                fontSize: '14px',
                                cursor: isSaving ? 'not-allowed' : 'pointer'
                            }}
                        >
                            Cancel
                        </button>
                    </div>
                </div>
            ) : (
                <div style={{
                    background: 'rgba(30, 41, 59, 0.3)',
                    border: '1px solid #334155',
                    borderRadius: 8,
                    minHeight: '100px'
                }}>
                    {renderMarkdown(readme)}
                </div>
            )}
        </div>
    );
}