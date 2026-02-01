interface HeaderProps {
    dir: string;
    onDirChange: (dir: string) => void;
    loading: boolean;
    onRefresh: () => void;
}

export function Header({ 
    dir, 
    onDirChange, 
    loading, 
    onRefresh, 
}: HeaderProps) {
    return (
        <div style={{ 
            padding: '12px 20px', 
            borderBottom: '1px solid #e5e5e5',
            backgroundColor: '#f9fafb',
            display: 'flex',
            alignItems: 'center',
            gap: '12px',
        }}>
            <h1 style={{ margin: 0, fontSize: '18px', fontWeight: 600 }}>Code Review</h1>
            <input
                type="text"
                value={dir}
                onChange={(e) => onDirChange(e.target.value)}
                placeholder="Directory path"
                style={{
                    flex: 1,
                    maxWidth: '500px',
                    padding: '8px 12px',
                    fontSize: '13px',
                    border: '1px solid #d1d5db',
                    borderRadius: '6px',
                }}
            />
            <button
                onClick={onRefresh}
                disabled={loading}
                style={{
                    padding: '8px 16px',
                    fontSize: '13px',
                    backgroundColor: '#4CAF50',
                    color: 'white',
                    border: 'none',
                    borderRadius: '6px',
                    cursor: loading ? 'not-allowed' : 'pointer',
                    opacity: loading ? 0.7 : 1,
                }}
            >
                {loading ? 'Loading...' : 'Refresh'}
            </button>
        </div>
    );
}
