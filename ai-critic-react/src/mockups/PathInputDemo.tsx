import { useState } from 'react';
import { MockupPageContainer } from './MockupPageContainer';
import { PathInput } from '../pure-view/PathInput';
import './PathInputDemo.css';

export function PathInputDemo() {
    const [path1, setPath1] = useState('/workspace/feature-request-app');
    const [path2, setPath2] = useState('/home/user/projects/my-awesome-project/with-very-very-very-long-name-here');
    const [path3, setPath3] = useState('/workspace/backend/api/v2/modules/auth/handlers/middleware/services/repositories/database/connection/pool');

    return (
        <MockupPageContainer
            title="PathInput"
            description="Editable path input with no-zoom for iOS"
        >
            <div className="path-input-demo">
                <div className="path-input-section">
                    <h3>Short Path (1 row)</h3>
                    <PathInput
                        value={path1}
                        onChange={setPath1}
                        label="Project Directory"
                    />
                </div>

                <div className="path-input-section">
                    <h3>Long Path (2 rows)</h3>
                    <PathInput
                        value={path2}
                        onChange={setPath2}
                        label="Long Path"
                    />
                </div>

                <div className="path-input-section">
                    <h3>Very Long Path (3 rows)</h3>
                    <PathInput
                        value={path3}
                        onChange={setPath3}
                        label="Nested Directory"
                    />
                </div>
            </div>
        </MockupPageContainer>
    );
}
