export type FlowStep = 'understanding' | 'clarifying' | 'implementing' | 'verifying' | 'completed';

export interface FlowStatus {
    step: FlowStep;
    title: string;
    description: string;
    status: 'pending' | 'active' | 'completed' | 'error';
}

export type InsightType = 'decision' | 'action' | 'question';

export interface Insight {
    id: string;
    type: InsightType;
    title: string;
    description: string;
    agent: string;
}

export type DriverAgentStatus = 'idle' | 'running' | 'paused' | 'finished';

export interface FeatureMakerContentProps {
    initialProjectDir?: string;
    featureTitle?: string;
    featureDescription?: string;
    onFeatureTitleChange?: (title: string) => void;
    onFeatureDescriptionChange?: (description: string) => void;
    useRealData?: boolean;
}
