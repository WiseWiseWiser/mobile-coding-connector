export type AuthUserMode = 'any' | 'fixed';
export type AuthTokenMode = 'shared' | 'custom';

export interface ServiceAuthFieldsProps {
    requireAuth: boolean;
    authUserMode: AuthUserMode;
    authUser: string;
    authTokenMode: AuthTokenMode;
    authToken: string;
    onRequireAuthChange: (value: boolean) => void;
    onAuthUserModeChange: (value: AuthUserMode) => void;
    onAuthUserChange: (value: string) => void;
    onAuthTokenModeChange: (value: AuthTokenMode) => void;
    onAuthTokenChange: (value: string) => void;
}

export function ServiceAuthFields({
    requireAuth,
    authUserMode,
    authUser,
    authTokenMode,
    authToken,
    onRequireAuthChange,
    onAuthUserModeChange,
    onAuthUserChange,
    onAuthTokenModeChange,
    onAuthTokenChange,
}: ServiceAuthFieldsProps) {
    return (
        <>
            <button
                type="button"
                className={`mcc-service-toggle ${requireAuth ? 'active' : ''}`}
                onClick={() => onRequireAuthChange(!requireAuth)}
            >
                {requireAuth ? 'Require Auth enabled' : 'Require Auth'}
            </button>
            {requireAuth && (
                <div className="mcc-service-auth-fields">
                    <div className="mcc-form-field">
                        <label>Auth User</label>
                        <div className="mcc-provider-options">
                            <button
                                type="button"
                                className={`mcc-provider-btn ${authUserMode === 'any' ? 'active' : ''}`}
                                onClick={() => onAuthUserModeChange('any')}
                            >
                                Any user
                            </button>
                            <button
                                type="button"
                                className={`mcc-provider-btn ${authUserMode === 'fixed' ? 'active' : ''}`}
                                onClick={() => onAuthUserModeChange('fixed')}
                            >
                                Fixed
                            </button>
                        </div>
                    </div>
                    {authUserMode === 'fixed' && (
                        <div className="mcc-form-field">
                            <label>Username</label>
                            <input
                                type="text"
                                placeholder="alice"
                                value={authUser}
                                onChange={(e) => onAuthUserChange(e.target.value)}
                            />
                        </div>
                    )}
                    <div className="mcc-form-field">
                        <label>Auth Token</label>
                        <div className="mcc-provider-options">
                            <button
                                type="button"
                                className={`mcc-provider-btn ${authTokenMode === 'shared' ? 'active' : ''}`}
                                onClick={() => onAuthTokenModeChange('shared')}
                            >
                                Shared server token
                            </button>
                            <button
                                type="button"
                                className={`mcc-provider-btn ${authTokenMode === 'custom' ? 'active' : ''}`}
                                onClick={() => onAuthTokenModeChange('custom')}
                            >
                                Custom
                            </button>
                        </div>
                    </div>
                    {authTokenMode === 'custom' && (
                        <div className="mcc-form-field">
                            <label>Token</label>
                            <input
                                type="password"
                                placeholder="token"
                                value={authToken}
                                onChange={(e) => onAuthTokenChange(e.target.value)}
                            />
                        </div>
                    )}
                </div>
            )}
        </>
    );
}
