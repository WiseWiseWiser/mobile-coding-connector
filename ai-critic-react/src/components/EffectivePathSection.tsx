// EffectivePathSection.tsx
// Add this component to your diagnose page above the refresh button

import React, { useState, useEffect } from 'react';

interface PathInfoResponse {
  system_path: string;
  user_paths: string[];
  extra_paths: string[];
  first_pass_path: string;
  second_pass_path: string;
  final_path: string;
  node_installations: Array<{
    path: string;
    version: string;
    dir: string;
  }>;
}

export const EffectivePathSection: React.FC = () => {
  const [pathInfo, setPathInfo] = useState<PathInfoResponse | null>(null);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);

  const fetchPathInfo = async () => {
    setLoading(true);
    setError(null);
    try {
      const response = await fetch('/api/tools/path-info');
      console.log('Path info API response:', response.status, response.statusText);
      if (!response.ok) {
        throw new Error(`HTTP error! status: ${response.status}`);
      }
      const data = await response.json();
      console.log('Path info data:', data);
      setPathInfo(data);
    } catch (err) {
      console.error('Path info fetch error:', err);
      setError(err instanceof Error ? err.message : 'Failed to fetch');
    } finally {
      setLoading(false);
    }
  };

  useEffect(() => {
    fetchPathInfo();
  }, []);

  if (loading) return <div style={{ padding: 16, color: 'var(--mcc-text-secondary)' }}>Loading path info...</div>;
  if (error) return <div style={{ padding: 16, color: 'var(--mcc-error-color)' }}>Error: {error}</div>;
  if (!pathInfo) return null;

  const formatPath = (pathStr: string) => pathStr ? pathStr.split(':') : [];

  return (
    <div style={{ 
      padding: '16px', 
      backgroundColor: 'var(--mcc-bg-card)', 
      borderRadius: '8px',
      marginBottom: '16px',
      border: '1px solid var(--mcc-border-color)'
    }}>
      <h2 style={{ marginTop: 0, marginBottom: '16px', color: 'var(--mcc-text-primary)' }}>Effective Path</h2>
      
      {/* System PATH */}
      <div style={{ marginBottom: '12px' }}>
        <strong style={{ color: 'var(--mcc-text-primary)' }}>1. System PATH:</strong>
        <div style={{ 
          backgroundColor: 'var(--mcc-bg-secondary)', 
          padding: '8px', 
          borderRadius: '4px',
          fontFamily: 'monospace',
          fontSize: '12px',
          maxHeight: '100px',
          overflowY: 'auto',
          color: 'var(--mcc-text-primary)'
        }}>
          {formatPath(pathInfo.system_path).map((p, i) => (
            <div key={i}>{p}</div>
          ))}
        </div>
      </div>

      {/* User Paths */}
      <div style={{ marginBottom: '12px' }}>
        <strong style={{ color: 'var(--mcc-text-primary)' }}>2. User Configured Paths:</strong>
        <div style={{ 
          backgroundColor: 'var(--mcc-bg-secondary)', 
          padding: '8px', 
          borderRadius: '4px',
          fontFamily: 'monospace',
          fontSize: '12px',
          color: 'var(--mcc-text-primary)'
        }}>
          {pathInfo.user_paths && pathInfo.user_paths.length > 0 ? (
            pathInfo.user_paths.map((p, i) => (
              <div key={i}>{p}</div>
            ))
          ) : (
            <em style={{ color: 'var(--mcc-text-muted)' }}>No user paths configured</em>
          )}
        </div>
      </div>

      {/* Extra Paths */}
      <div style={{ marginBottom: '12px' }}>
        <strong style={{ color: 'var(--mcc-text-primary)' }}>3. Extra Paths (npm, node, etc.):</strong>
        <div style={{ 
          backgroundColor: 'var(--mcc-bg-secondary)', 
          padding: '8px', 
          borderRadius: '4px',
          fontFamily: 'monospace',
          fontSize: '12px',
          maxHeight: '150px',
          overflowY: 'auto',
          color: 'var(--mcc-text-primary)'
        }}>
          {pathInfo.extra_paths && pathInfo.extra_paths.length > 0 ? (
            pathInfo.extra_paths.map((p, i) => (
              <div key={i}>{p}</div>
            ))
          ) : (
            <em style={{ color: 'var(--mcc-text-muted)' }}>No extra paths</em>
          )}
        </div>
      </div>

      {/* First Pass */}
      <div style={{ marginBottom: '12px' }}>
        <strong style={{ color: 'var(--mcc-text-primary)' }}>4. First Pass (System + User + Extra):</strong>
        <div style={{ 
          backgroundColor: '#1e3a5f', 
          padding: '8px', 
          borderRadius: '4px',
          fontFamily: 'monospace',
          fontSize: '12px',
          maxHeight: '100px',
          overflowY: 'auto',
          color: 'var(--mcc-text-primary)'
        }}>
          {formatPath(pathInfo.first_pass_path).map((p, i) => (
            <div key={i}>{p}</div>
          ))}
        </div>
      </div>

      {/* Second Pass */}
      <div style={{ marginBottom: '12px' }}>
        <strong style={{ color: 'var(--mcc-text-primary)' }}>5. Second Pass (Prioritized by Node Version):</strong>
        <div style={{ 
          backgroundColor: '#3d2a1e', 
          padding: '8px', 
          borderRadius: '4px',
          fontFamily: 'monospace',
          fontSize: '12px',
          maxHeight: '100px',
          overflowY: 'auto',
          color: 'var(--mcc-text-primary)'
        }}>
          {formatPath(pathInfo.second_pass_path).map((p, i) => (
            <div key={i}>{p}</div>
          ))}
        </div>
      </div>

      {/* Final PATH */}
      <div style={{ marginBottom: '12px' }}>
        <strong style={{ color: 'var(--mcc-text-primary)' }}>6. Final PATH:</strong>
        <div style={{ 
          backgroundColor: '#1a3d2e', 
          padding: '8px', 
          borderRadius: '4px',
          fontFamily: 'monospace',
          fontSize: '12px',
          maxHeight: '100px',
          overflowY: 'auto',
          border: '2px solid var(--mcc-success-color)',
          color: 'var(--mcc-text-primary)'
        }}>
          {formatPath(pathInfo.final_path).map((p, i) => (
            <div key={i}>{p}</div>
          ))}
        </div>
      </div>

      {/* Node Installations Summary */}
      {pathInfo.node_installations && pathInfo.node_installations.length > 0 && (
        <div style={{ 
          marginTop: '16px', 
          padding: '12px', 
          backgroundColor: 'var(--mcc-bg-secondary)', 
          borderRadius: '4px',
          border: '1px solid var(--mcc-accent-color)'
        }}>
          <h4 style={{ marginTop: 0, marginBottom: '12px', color: 'var(--mcc-accent-color)' }}>
            Discovered Node Installations
          </h4>
          <table style={{ width: '100%', borderCollapse: 'collapse', fontSize: '13px' }}>
            <thead>
              <tr style={{ backgroundColor: 'var(--mcc-bg-card)' }}>
                <th style={{ padding: '8px', textAlign: 'left', borderBottom: '2px solid var(--mcc-border-color)', color: 'var(--mcc-text-primary)' }}>Version</th>
                <th style={{ padding: '8px', textAlign: 'left', borderBottom: '2px solid var(--mcc-border-color)', color: 'var(--mcc-text-primary)' }}>Directory</th>
                <th style={{ padding: '8px', textAlign: 'left', borderBottom: '2px solid var(--mcc-border-color)', color: 'var(--mcc-text-primary)' }}>Full Path</th>
              </tr>
            </thead>
            <tbody>
              {pathInfo.node_installations.map((node, idx) => (
                <tr key={idx} style={{ backgroundColor: idx % 2 === 0 ? 'var(--mcc-bg-card)' : 'var(--mcc-bg-secondary)' }}>
                  <td style={{ padding: '8px', borderBottom: '1px solid var(--mcc-border-color)', fontFamily: 'monospace', color: 'var(--mcc-text-primary)' }}>
                    <strong>{node.version}</strong>
                  </td>
                  <td style={{ padding: '8px', borderBottom: '1px solid var(--mcc-border-color)', fontSize: '12px', color: 'var(--mcc-text-secondary)' }}>
                    {node.dir}
                  </td>
                  <td style={{ padding: '8px', borderBottom: '1px solid var(--mcc-border-color)', fontFamily: 'monospace', fontSize: '11px', color: 'var(--mcc-text-muted)' }}>
                    {node.path}
                  </td>
                </tr>
              ))}
            </tbody>
          </table>
        </div>
      )}

      {/* Refresh Button */}
      <div style={{ marginTop: '20px', textAlign: 'center' }}>
        <button
          onClick={fetchPathInfo}
          disabled={loading}
          style={{
            padding: '12px 32px',
            backgroundColor: loading ? 'var(--mcc-text-muted)' : 'var(--mcc-accent-color)',
            color: 'white',
            border: 'none',
            borderRadius: '4px',
            cursor: loading ? 'not-allowed' : 'pointer',
            fontSize: '14px',
            fontWeight: 'bold',
            boxShadow: '0 2px 4px rgba(0,0,0,0.2)'
          }}
        >
          {loading ? 'Loading...' : 'Refresh Path Info'}
        </button>
      </div>
    </div>
  );
};

export default EffectivePathSection;
