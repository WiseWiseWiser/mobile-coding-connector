// Navigation tabs
export const NavTabs = {
    Home: 'home',
    Agent: 'agent',
    Terminal: 'terminal',
    Ports: 'ports',
    Files: 'files',
} as const;

export type NavTab = typeof NavTabs[keyof typeof NavTabs];
