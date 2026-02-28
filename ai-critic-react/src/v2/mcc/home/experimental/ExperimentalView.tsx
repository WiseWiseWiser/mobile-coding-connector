import { useNavigate } from 'react-router-dom';
import { BeakerIcon } from '../../../icons';

export function ExperimentalView() {
    const navigate = useNavigate();

    return (
        <div className="mcc-experimental-container">
            <div className="mcc-section-header">
                <button className="mcc-back-btn" onClick={() => navigate('..')}>&larr;</button>
                <BeakerIcon className="mcc-header-icon" />
                <h2>Experimental Features</h2>
            </div>
            <p className="mcc-section-subtitle mcc-experimental-subtitle">
                Try out new and experimental features. These may be unstable or change frequently.
            </p>

            <div className="mcc-experimental-cards">
                <div
                    className="mcc-experimental-card mcc-experimental-card-clickable"
                    onClick={() => navigate('../codex-web')}
                >
                    <div className="mcc-experimental-card-icon">
                        <BeakerIcon />
                    </div>
                    <div className="mcc-experimental-card-content">
                        <h3>Codex Web</h3>
                        <p>
                            Web interface for OpenAI Codex CLI. Run Codex commands through a web UI.
                        </p>
                        <span className="mcc-experimental-status mcc-experimental-status-active">Click to Open</span>
                    </div>
                </div>
                <div
                    className="mcc-experimental-card mcc-experimental-card-clickable"
                    onClick={() => navigate('../cursor-web')}
                >
                    <div className="mcc-experimental-card-icon">
                        <BeakerIcon />
                    </div>
                    <div className="mcc-experimental-card-content">
                        <h3>Cursor Web</h3>
                        <p>
                            Web interface powered by Cloud CLI (siteboon/claudecodeui) for Cursor CLI workflows.
                        </p>
                        <span className="mcc-experimental-status mcc-experimental-status-active">Click to Open</span>
                    </div>
                </div>
                <div
                    className="mcc-experimental-card mcc-experimental-card-clickable"
                    onClick={() => navigate('../opencode-web')}
                >
                    <div className="mcc-experimental-card-icon">
                        <BeakerIcon />
                    </div>
                    <div className="mcc-experimental-card-content">
                        <h3>OpenCode Web</h3>
                        <p>
                            Web interface for the existing exposed OpenCode server managed by this system.
                        </p>
                        <span className="mcc-experimental-status mcc-experimental-status-active">Click to Open</span>
                    </div>
                </div>
            </div>
        </div>
    );
}
