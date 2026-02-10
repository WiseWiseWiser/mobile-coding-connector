import { useMemo } from 'react';
import { loadSSHKeys } from '../v2/mcc/home/settings/gitStorage';
import type { ProjectInfo } from '../api/projects';

export interface SSHKeyValidationResult {
    hasSSHKey: boolean;
    isKeyAvailable: boolean;
    message: string;
}

/**
 * Validates if a project has a properly configured SSH key.
 * Returns validation result with status and message.
 */
export function validateProjectSSHKey(project: ProjectInfo | undefined): SSHKeyValidationResult {
    if (!project) {
        return {
            hasSSHKey: false,
            isKeyAvailable: false,
            message: 'No project selected',
        };
    }

    if (!project.ssh_key_id) {
        return {
            hasSSHKey: false,
            isKeyAvailable: false,
            message: 'SSH key required for this operation. Configure in project settings.',
        };
    }

    // Verify the key still exists in storage
    const sshKeys = loadSSHKeys();
    const keyExists = sshKeys.some(k => k.id === project.ssh_key_id);

    if (!keyExists) {
        return {
            hasSSHKey: true,
            isKeyAvailable: false,
            message: 'Configured SSH key not found. Please reconfigure in project settings.',
        };
    }

    return {
        hasSSHKey: true,
        isKeyAvailable: true,
        message: '',
    };
}

/**
 * Hook to check SSH key validation status for a project.
 * Memoized to prevent unnecessary recalculations.
 */
export function useSSHKeyValidation(project: ProjectInfo | undefined): SSHKeyValidationResult {
    return useMemo(() => validateProjectSSHKey(project), [project?.ssh_key_id, project?.use_ssh]);
}
