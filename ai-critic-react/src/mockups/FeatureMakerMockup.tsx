import { MockupPageContainer } from './MockupPageContainer';
import { FeatureMakerContent } from '../components/feature-maker';

export { FeatureMakerContent };

export function FeatureMakerMockup() {
    return (
        <MockupPageContainer
            title="FeatureMaker"
            description="AI-driven feature implementation flow with subagents"
        >
            <FeatureMakerContent />
        </MockupPageContainer>
    );
}
