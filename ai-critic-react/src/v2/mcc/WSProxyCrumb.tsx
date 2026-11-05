import { useNavigate } from 'react-router-dom';

export function WSProxyCrumb({ parts }: { parts: { label: string; view?: string }[] }) {
    const navigate = useNavigate();
    return (
        <div className="mcc-crumb">
            {parts.map((p, i) => (
                <span key={i}>
                    {i > 0 && <span className="mcc-crumb-sep">/</span>}
                    {p.view !== undefined
                        ? <a className="mcc-crumb-link" onClick={() => navigate(p.view!, { relative: 'path' })}>{p.label}</a>
                        : <span>{p.label}</span>}
                </span>
            ))}
        </div>
    );
}
