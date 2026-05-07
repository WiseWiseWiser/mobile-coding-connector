import { useNavigate } from 'react-router-dom';
import { useProjectDir } from '../../../../hooks/project/useProjectDir';
import { useV2Context } from '../../../V2Context';
import { CodexCliChat } from '../../agent/CodexCliChat';

export interface CodexWebUIProps {
    backPath?: string;
}

export function CodexWebUI({ backPath = '../experimental' }: CodexWebUIProps) {
    const navigate = useNavigate();
    const { currentProject } = useV2Context();
    const { projectDir } = useProjectDir();

    if (!projectDir) {
        return (
            <div className="mcc-agent-view">
                <div className="mcc-agent-error">No project directory selected.</div>
            </div>
        );
    }

    return (
        <CodexCliChat
            projectName={currentProject?.name ?? null}
            projectDir={projectDir}
            onBack={() => navigate(backPath)}
            onSettings={() => navigate('settings')}
        />
    );
}
