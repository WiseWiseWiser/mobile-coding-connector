import { useState, useRef, useEffect } from 'react';
import { getFakeLLMServer, type FakeLLMSession, type StreamEvent } from './fake';
import type { FlowStep, Insight, DriverAgentStatus } from '../components/feature-maker/types';
import { flowSteps, mockFeatureRequest } from './featureMakerMockData';
import { FlowStages } from '../components/feature-maker/FlowStages';
import { AgentSidebar } from '../components/feature-maker/AgentSidebar';
import { DriverSection } from '../components/feature-maker/DriverSection';
import { FlowPanels } from '../components/feature-maker/FlowPanels';
import { InsightModal } from '../components/feature-maker/InsightModal';
import '../components/feature-maker/FeatureMakerContent.css';
import '../components/feature-maker/FeatureRequestSection.css';

export function FeatureMakerMockupContent() {
    const [currentStep] = useState<FlowStep>('clarifying');
    const [showDoc, setShowDoc] = useState(false);
    const [selectedInsight, setSelectedInsight] = useState<Insight | null>(null);
    const [projectPath, setProjectPath] = useState('/workspace/feature-request-app');
    const [driverAgentStatus, setDriverAgentStatus] = useState<DriverAgentStatus>('idle');
    const [streamEvents, setStreamEvents] = useState<StreamEvent[]>([]);
    const [progress, setProgress] = useState(0);
    const sessionRef = useRef<FakeLLMSession | null>(null);
    const streamEventsRef = useRef<HTMLDivElement | null>(null);

    useEffect(() => {
        if (streamEventsRef.current) {
            streamEventsRef.current.scrollTop = streamEventsRef.current.scrollHeight;
        }
    }, [streamEvents]);

    const handleStart = () => {
        setDriverAgentStatus('running');
        setStreamEvents([]);
        setProgress(0);

        const server = getFakeLLMServer();
        const session = server.startStream(mockFeatureRequest.description, projectPath);
        sessionRef.current = session;

        session.onStart(() => {
            setStreamEvents(prev => [...prev, { type: 'start', message: 'Starting driver agent...', timestamp: Date.now() }]);
        });

        session.onStep((event) => {
            setStreamEvents(prev => [...prev, event]);
            if (event.progress) setProgress(event.progress);
        });

        session.onDone((event) => {
            setStreamEvents(prev => [...prev, event]);
            setDriverAgentStatus('finished');
            setProgress(100);
            sessionRef.current = null;
        });

        session.onAborted((event) => {
            setStreamEvents(prev => [...prev, event]);
            setDriverAgentStatus('idle');
            sessionRef.current = null;
        });
    };

    const handlePause = () => setDriverAgentStatus('paused');

    const handleResume = () => {
        setDriverAgentStatus('running');
    };

    const handleAbort = () => {
        if (sessionRef.current) {
            sessionRef.current.abort();
            sessionRef.current = null;
        }
        setDriverAgentStatus('idle');
        setProgress(0);
    };

    return (
        <div className="feature-maker">
            <div className="fm-header">
                <button
                    className={`fm-toggle-doc ${showDoc ? 'active' : ''}`}
                    onClick={() => setShowDoc(!showDoc)}
                >
                    {showDoc ? '▼' : '▶'} Flow Documentation
                </button>
            </div>

            {showDoc && (
                <div className="fm-doc-panel">
                    <h3>FeatureMaker Flow</h3>
                    <p>
                        FeatureMaker is an AI-driven workflow for implementing features using a driver agent
                        that orchestrates dedicated subagents. The flow follows a 4-step process:
                    </p>
                    <ol>
                        <li>
                            <strong>Understanding the Request</strong> - Parse and comprehend the feature request,
                            identify goals, constraints, and success criteria
                        </li>
                        <li>
                            <strong>Clarify Architectural Decisions</strong> - Determine API contracts, data models,
                            integration points, and technical approach
                        </li>
                        <li>
                            <strong>Implementing with Coding</strong> - Write code, create components,
                            and integrate with existing systems using subagents
                        </li>
                        <li>
                            <strong>Verify the Implementation</strong> - Run tests, validate against requirements,
                            ensure code quality and performance
                        </li>
                    </ol>
                    <p className="fm-doc-note">
                        A <strong>Driver Agent</strong> orchestrates this flow, delegating subprocesses to
                        dedicated subagents (Architect, Coder, Verifier).
                    </p>
                </div>
            )}

            <div className="fm-content">
                <div className="fm-feature-request">
                    <div className="fm-fr-header">
                        <h3>Feature Request</h3>
                        <div className="fm-fr-meta">
                            <span className={`fm-priority ${mockFeatureRequest.priority}`}>
                                {mockFeatureRequest.priority}
                            </span>
                            <span className="fm-complexity">
                                Complexity: {mockFeatureRequest.estimatedComplexity}
                            </span>
                        </div>
                    </div>
                    <div className="fm-fr-content">
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
                    </div>
                </div>

                <FlowStages steps={flowSteps} currentStep={currentStep} />

                <AgentSidebar
                    showPathInput
                    projectPath={projectPath}
                    onProjectPathChange={setProjectPath}
                />

                <div className="fm-panels">
                    <DriverSection
                        status={driverAgentStatus}
                        progress={progress}
                        streamEvents={streamEvents}
                        streamEventsRef={streamEventsRef}
                        onStart={handleStart}
                        onPause={handlePause}
                        onResume={handleResume}
                        onAbort={handleAbort}
                    />

                    <FlowPanels
                        selectedInsight={selectedInsight}
                        onSelectInsight={setSelectedInsight}
                    />
                </div>

                {selectedInsight && (
                    <InsightModal
                        insight={selectedInsight}
                        onClose={() => setSelectedInsight(null)}
                    />
                )}
            </div>
        </div>
    );
}
