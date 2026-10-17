import { useCallback, useEffect, useState } from 'react';
import type { JSX } from 'react';

import { api, setCSRF } from '../lib/api.js';
import { errorMessage, isCancelled } from '../lib/i18n.js';
import type { Page, Session, Status } from '../lib/types.js';
import { Auth } from './Auth.js';
import { Feedback } from './FormField.js';
import { usePreferences } from './Preferences.js';
import { Workspace } from './Workspace.js';

export function App(): JSX.Element {
    const { t } = usePreferences();
    const [error, setError] = useState('');
    const [page, setPage] = useState<Page | null>(null);
    const [version, setVersion] = useState('');
    const refresh = useCallback(async (signal?: AbortSignal) => {
        const session = await api<Session>(
            '/api/session',
            'GET',
            undefined,
            signal
        );
        setCSRF(session.csrf);
        let next: Page = session.initialized
            ? session.authenticated
                ? 'app'
                : 'login'
            : 'setup';
        if (
            session.initialized &&
            (globalThis.location.pathname.endsWith('/forgot-password') ||
                document.querySelector<HTMLElement>('#root')?.dataset.page ===
                    'recovery')
        ) {
            next = 'recovery';
        } else if (session.authenticated) {
            const status = await api<Status>(
                '/api/settings/pikpak',
                'GET',
                undefined,
                signal
            );
            if (!status.connected && !status.source) {
                next = 'onboarding';
            }
        }
        if (signal?.aborted) {
            return;
        }
        setPage(next);
        setVersion(session.version);
        setError('');
    }, []);
    const onAuthenticated = async () => {
        await refresh();
    };
    useEffect(() => {
        const controller = new AbortController();
        // The API response updates state asynchronously; cleanup cancels the request.
        // eslint-disable-next-line react-hooks/set-state-in-effect
        void refresh(controller.signal).catch((error_: unknown) => {
            if (!controller.signal.aborted && !isCancelled(error_)) {
                setError(errorMessage(error_));
            }
        });
        const expired = () => {
            setPage('login');
        };
        globalThis.addEventListener('session-expired', expired);
        return () => {
            controller.abort();
            globalThis.removeEventListener('session-expired', expired);
        };
    }, [refresh]);
    if (!page) {
        return (
            <main className='login-shell'>
                <Feedback message={t(error)} />
                {error && (
                    <button
                        className='button subtle'
                        onClick={() => {
                            void refresh().catch((error_: unknown) => {
                                setError(errorMessage(error_));
                            });
                        }}
                        type='button'
                    >
                        {t('重新整理')}
                    </button>
                )}
            </main>
        );
    }
    return page === 'app' ? (
        <Workspace onAuthenticated={onAuthenticated} version={version} />
    ) : (
        <Auth
            key={page}
            onAuthenticated={onAuthenticated}
            page={page}
            version={version}
        />
    );
}
