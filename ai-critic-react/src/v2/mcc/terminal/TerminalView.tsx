import { TerminalManager } from './TerminalManager';

export function TerminalView() {
    return (
        <div className="mcc-terminal-container">
            <TerminalManager isVisible={true} />
        </div>
    );
}
