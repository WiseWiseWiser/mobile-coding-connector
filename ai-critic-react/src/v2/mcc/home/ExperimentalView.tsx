import { useNavigate } from 'react-router-dom';
import { BeakerIcon } from '../../icons';

export function ExperimentalView() {
    const navigate = useNavigate();

    return (
        <div className="mcc-experimental-container">
            <div className="mcc-section-header">
                <h2>Experimental Features</h2>
                <p className="mcc-section-subtitle">
                    Try out new and experimental features. These may be unstable or change frequently.
                </p>
            </div>

            <div className="mcc-experimental-cards">
                <div 
                    className="mcc-experimental-card mcc-experimental-card-clickable"
                    onClick={() => navigate('/home/codex-web')}
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
            </div>

            <button 
                className="mcc-back-btn"
                onClick={() => navigate('/home')}
            >
                Back to Home
            </button>
        </div>
    );
}
