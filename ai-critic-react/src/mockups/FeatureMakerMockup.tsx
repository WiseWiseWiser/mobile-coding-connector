import { MockupPageContainer } from './MockupPageContainer';
import { FeatureMakerMockupContent } from './FeatureMakerMockupContent';

export function FeatureMakerMockup() {
    return (
        <MockupPageContainer
            title="FeatureMaker"
            description="AI-driven feature implementation flow with subagents"
        >
            <FeatureMakerMockupContent />
        </MockupPageContainer>
    );
}
