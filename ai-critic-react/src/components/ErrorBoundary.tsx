import React from 'react';

interface ErrorBoundaryProps {
    children: React.ReactNode;
    fallback?: React.ReactNode;
}

interface ErrorBoundaryState {
    hasError: boolean;
    error: Error | null;
}

export class ErrorBoundary extends React.Component<ErrorBoundaryProps, ErrorBoundaryState> {
    constructor(props: ErrorBoundaryProps) {
        super(props);
        this.state = {
            hasError: false,
            error: null
        };
    }

    static getDerivedStateFromError(error: Error): ErrorBoundaryState {
        return {
            hasError: true,
            error
        };
    }

    componentDidCatch(error: Error, errorInfo: React.ErrorInfo) {
        console.error('ErrorBoundary caught an error:', error, errorInfo);
    }

    render() {
        if (this.state.hasError) {
            if (this.props.fallback) {
                return this.props.fallback;
            }

            return (
                <div style={{
                    padding: '20px',
                    textAlign: 'center',
                    color: '#e2e8f0',
                    background: '#1e293b',
                    border: '1px solid #ef4444',
                    borderRadius: '8px',
                    margin: '16px'
                }}>
                    <div style={{ fontSize: '20px', fontWeight: 600, marginBottom: '12px', color: '#ef4444' }}>
                        ⚠️ Something went wrong
                    </div>
                    <div style={{ fontSize: '14px', marginBottom: '16px', color: '#94a3b8' }}>
                        An error occurred while rendering this component
                    </div>
                    {this.state.error && (
                        <div style={{
                            padding: '12px',
                            background: 'rgba(239, 68, 68, 0.1)',
                            border: '1px solid rgba(239, 68, 68, 0.3)',
                            borderRadius: '6px',
                            fontSize: '12px',
                            color: '#fca5a5',
                            textAlign: 'left',
                            wordBreak: 'break-word',
                            whiteSpace: 'pre-wrap'
                        }}>
                            {this.state.error.toString()}
                        </div>
                    )}
                    <button
                        onClick={() => window.location.reload()}
                        style={{
                            marginTop: '16px',
                            padding: '10px 20px',
                            background: '#3b82f6',
                            color: '#fff',
                            border: 'none',
                            borderRadius: '6px',
                            fontSize: '14px',
                            fontWeight: 600,
                            cursor: 'pointer'
                        }}
                    >
                        Reload Page
                    </button>
                    <button
                        onClick={() => this.setState({ hasError: false, error: null })}
                        style={{
                            marginTop: '16px',
                            marginLeft: '10px',
                            padding: '10px 20px',
                            background: '#1e293b',
                            color: '#e2e8f0',
                            border: '1px solid #334155',
                            borderRadius: '6px',
                            fontSize: '14px',
                            fontWeight: 600,
                            cursor: 'pointer'
                        }}
                    >
                        Dismiss
                    </button>
                </div>
            );
        }

        return this.props.children;
    }
}
