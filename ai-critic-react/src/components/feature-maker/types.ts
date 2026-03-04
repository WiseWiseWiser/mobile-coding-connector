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
    featureTitle?: string;
    featureDescription?: string;
    editing?: boolean;
    editTitle?: string;
    editDescription?: string;
    onEditTitleChange?: (title: string) => void;
    onEditDescriptionChange?: (description: string) => void;
    onEdit?: () => void;
    onSave?: () => void;
    onCancel?: () => void;
}
