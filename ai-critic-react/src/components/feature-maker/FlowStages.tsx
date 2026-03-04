import type { FlowStep, FlowStatus } from './types';
import './FlowStages.css';

interface FlowStagesProps {
    steps: FlowStatus[];
    currentStep: FlowStep;
}

export function FlowStages({ steps, currentStep }: FlowStagesProps) {
    return (
        <div className="fm-flow-stages">
            {steps.map((step, index) => (
                <div
                    key={step.step}
                    className={`fm-flow-stage ${step.status} ${step.step === currentStep ? 'current' : ''}`}
                >
                    <div className="fm-stage-indicator">
                        {step.status === 'completed' ? '✓' : step.status === 'error' ? '✗' : index + 1}
                    </div>
                    <span className="fm-stage-title">{step.title}</span>
                </div>
            ))}
        </div>
    );
}
