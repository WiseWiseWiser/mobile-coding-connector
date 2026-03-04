import { mockFeatureRequest } from './mockData';

interface FeatureRequestSectionProps {
    useRealData?: boolean;
    featureTitle?: string;
    featureDescription?: string;
    onFeatureTitleChange?: (title: string) => void;
    onFeatureDescriptionChange?: (description: string) => void;
}

export function FeatureRequestSection({
    useRealData,
    featureTitle,
    featureDescription,
    onFeatureTitleChange,
    onFeatureDescriptionChange,
}: FeatureRequestSectionProps) {
    return (
        <div className="fm-feature-request">
            <div className="fm-fr-header">
                <h3>Feature Request</h3>
                {!useRealData && (
                    <div className="fm-fr-meta">
                        <span className={`fm-priority ${mockFeatureRequest.priority}`}>
                            {mockFeatureRequest.priority}
                        </span>
                        <span className="fm-complexity">
                            Complexity: {mockFeatureRequest.estimatedComplexity}
                        </span>
                    </div>
                )}
            </div>
            <div className="fm-fr-content">
                {useRealData ? (
                    <>
                        <input
                            className="fm-fr-title-input"
                            value={featureTitle ?? ''}
                            onChange={e => onFeatureTitleChange?.(e.target.value)}
                            onBlur={e => onFeatureTitleChange?.(e.target.value)}
                            placeholder="Feature title"
                        />
                        <textarea
                            className="fm-fr-desc-input"
                            value={featureDescription ?? ''}
                            onChange={e => onFeatureDescriptionChange?.(e.target.value)}
                            onBlur={e => onFeatureDescriptionChange?.(e.target.value)}
                            placeholder="Feature description (optional)"
                            rows={3}
                        />
                    </>
                ) : (
                    <>
                        <h4>{mockFeatureRequest.title}</h4>
                        <p>{mockFeatureRequest.description}</p>
                        <div className="fm-fr-details">
                            <h5>Key Requirements:</h5>
                            <ul>
                                {mockFeatureRequest.details.map((detail, i) => (
                                    <li key={i}>{detail}</li>
                                ))}
                            </ul>
                        </div>
                    </>
                )}
            </div>
        </div>
    );
}
