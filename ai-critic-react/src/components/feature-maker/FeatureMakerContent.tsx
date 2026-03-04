import { useState, useRef, useEffect } from 'react';
import { getFakeLLMServer, type FakeLLMSession, type StreamEvent } from '../../mockups/fake';
import type { FlowStep, Insight, DriverAgentStatus, FeatureMakerContentProps } from './types';
import { flowSteps, mockFeatureRequest } from './mockData';
import { FeatureRequestSection } from './FeatureRequestSection';
import { FlowStages } from './FlowStages';
import { AgentSidebar } from './AgentSidebar';
import { DriverSection } from './DriverSection';
import { FlowPanels } from './FlowPanels';
import { InsightModal } from './InsightModal';
import '../../mockups/FeatureMakerMockup.css';

export function FeatureMakerContent({
    initialProjectDir,
    featureTitle,
    featureDescription,
    onFeatureTitleChange,
    onFeatureDescriptionChange,
    useRealData,
}: FeatureMakerContentProps = {}) {
    const [currentStep] = useState<FlowStep>('clarifying');
    const [showDoc, setShowDoc] = useState(false);
    const [selectedInsight, setSelectedInsight] = useState<Insight | null>(null);
    const [projectPath, setProjectPath] = useState(initialProjectDir || '/workspace/feature-request-app');
    const [driverAgentStatus, setDriverAgentStatus] = useState<DriverAgentStatus>('idle');
    const [streamEvents, setStreamEvents] = useState<StreamEvent[]>([]);
    const [progress, setProgress] = useState(0);
    const sessionRef = useRef<FakeLLMSession | null>(null);
    const streamEventsRef = useRef<HTMLDivElement | null>(null);

    useEffect(() => {
        if (initialProjectDir) {
            setProjectPath(initialProjectDir);
        }
    }, [initialProjectDir]);

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
            {initialProjectDir && (
                <div className="fm-project-dir-top">
                    <span className="fm-project-dir-label">Project Directory</span>
                    <span className="fm-project-dir-value">{projectPath}</span>
                </div>
            )}
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
                <FeatureRequestSection
                    useRealData={useRealData}
                    featureTitle={featureTitle}
                    featureDescription={featureDescription}
                    onFeatureTitleChange={onFeatureTitleChange}
                    onFeatureDescriptionChange={onFeatureDescriptionChange}
                />

                <FlowStages steps={flowSteps} currentStep={currentStep} />

                <AgentSidebar
                    showPathInput={!initialProjectDir}
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
