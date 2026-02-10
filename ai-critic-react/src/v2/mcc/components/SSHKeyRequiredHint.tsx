import React from 'react';

interface SSHKeyRequiredHintProps {
    message?: string;
    style?: React.CSSProperties;
}

/**
 * Reusable component to display SSH key requirement hint.
 * Shows a warning message when SSH key is not configured.
 */
export const SSHKeyRequiredHint: React.FC<SSHKeyRequiredHintProps> = ({
    message = 'SSH key required for this operation. Configure in project settings.',
    style,
}) => {
    return (
        <div
            style={{
                fontSize: '13px',
                color: '#f59e0b',
                padding: '10px 14px',
                background: 'rgba(245, 158, 11, 0.1)',
                border: '1px solid rgba(245, 158, 11, 0.3)',
                borderRadius: 8,
                marginBottom: 12,
                ...style,
            }}
        >
            {message}
        </div>
    );
};

export default SSHKeyRequiredHint;
