// TerminalView is now a placeholder component
// The actual TerminalManager is rendered persistently in MobileCodingConnector
// to preserve terminal state when switching tabs
export function TerminalView() {
    // Return null since the terminal is rendered at the layout level
    // and shown/hidden via CSS based on the active tab
    return null;
}
