// Navigation tabs
export const NavTabs = {
    Home: 'home',
    Agent: 'agent',
    Terminal: 'terminal',
    Service: 'service',
    Files: 'files',
    Experimental: 'experimental',
} as const;

export type NavTab = typeof NavTabs[keyof typeof NavTabs];
