import { useState } from 'react';
import { MockupPageContainer } from './MockupPageContainer';
import { PathInput } from '../pure-view/PathInput';
import './PathInputDemo.css';

export function PathInputDemo() {
    const [path, setPath] = useState('/workspace/feature-request-app');

    return (
        <MockupPageContainer
            title="PathInput"
            description="Editable path input with no-zoom for iOS"
        >
            <div className="path-input-demo">
                <PathInput
                    value={path}
                    onChange={setPath}
                    label="Project Directory"
                />
            </div>
        </MockupPageContainer>
    );
}
