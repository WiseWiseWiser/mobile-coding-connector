import { formatMarkdown } from './utils';

interface ReviewPanelProps {
    review: string | null;
}

export function ReviewPanel({ review }: ReviewPanelProps) {
    if (!review) {
        return null;
    }

    return (
        <div style={{ 
            height: '300px', 
            borderTop: '1px solid #e5e5e5',
            overflow: 'auto',
            backgroundColor: '#fff',
        }}>
            <div style={{ 
                padding: '12px 16px', 
                borderBottom: '1px solid #e5e5e5',
                fontWeight: 600,
                fontSize: '14px',
                position: 'sticky',
                top: 0,
                backgroundColor: '#fff',
            }}>
                AI Review
            </div>
            <div style={{ padding: '16px', lineHeight: 1.6, fontSize: '13px' }}>
                <div 
                    style={{ whiteSpace: 'pre-wrap' }}
                    dangerouslySetInnerHTML={{ __html: formatMarkdown(review) }}
                />
            </div>
        </div>
    );
}
