import { useState } from 'react';
import type { JSX } from 'react';

import { api } from '../lib/api.js';
import { errorMessage } from '../lib/i18n.js';
import { DirtyForm } from './DirtyForm.js';
import { Feedback, FormField } from './FormField.js';
import { Icon } from './Icon.js';
import { usePreferences } from './Preferences.js';

export function TokenForm({
    onboarding = false,
    onSaved,
    onCheck,
}: {
    onboarding?: boolean;
    onSaved: () => Promise<void>;
    onCheck?: () => Promise<void>;
}): JSX.Element {
    const { t } = usePreferences();
    const [error, setError] = useState('');
    const [pending, setPending] = useState('');
    const [token, setToken] = useState('');
    return (
        <form
            className={`${onboarding ? 'login-form ' : ''}form-stack`}
            data-onboarding={onboarding}
            id='token-form'
            onSubmit={(event) => {
                event.preventDefault();
                if (pending) {
                    return;
                }
                setPending('save');
                setError('');
                void api('/api/settings/pikpak', 'POST', { token })
                    .then(onSaved)
                    .catch((error_: unknown) => {
                        setError(errorMessage(error_));
                    })
                    .finally(() => {
                        setToken('');
                        setPending('');
                    });
            }}
        >
            <DirtyForm dirty={Boolean(token || pending)} />
            <FormField
                id='token'
                info={{
                    id: 'token-permissions',
                    label: t('PAT 權限說明'),
                    content: t('PAT 需具備「讀寫檔案」及「雲端下載」權限。'),
                }}
                label={t('個人存取權杖（PAT）')}
            >
                <input
                    autoComplete='off'
                    onChange={(event) => {
                        setToken(event.target.value);
                    }}
                    placeholder={t('貼上 PAT')}
                    required
                    type='password'
                    value={token}
                />
            </FormField>
            <p className='field-help pat-create-copy'>
                {t('還沒有 PAT 嗎？')}{' '}
                <a
                    className='text-link'
                    href='https://mypikpak.com/en-US/help-center/connected_apps/personal_access_tokens/create_personal_access_token'
                    rel='noopener noreferrer'
                    target='_blank'
                >
                    {t('建立 PAT')}
                </a>
            </p>
            {!onboarding && <Feedback message={t(error)} />}
            <div className={onboarding ? 'token-submit' : 'form-actions'}>
                <button
                    aria-busy={pending === 'save'}
                    className='button primary'
                    disabled={Boolean(pending) || !token.trim()}
                    id='save-token'
                    type='submit'
                >
                    {pending === 'save' && (
                        <Icon className='is-loading' name='loading' />
                    )}
                    {t(onboarding ? '驗證並完成設定' : '驗證並綁定')}
                </button>
                {onCheck && (
                    <button
                        aria-busy={pending === 'check'}
                        className='button subtle'
                        disabled={Boolean(pending)}
                        id='check-connection'
                        onClick={() => {
                            setPending('check');
                            setError('');
                            void onCheck()
                                .catch((error_: unknown) => {
                                    setError(errorMessage(error_));
                                })
                                .finally(() => {
                                    setPending('');
                                });
                        }}
                        type='button'
                    >
                        {pending === 'check' && (
                            <Icon className='is-loading' name='loading' />
                        )}
                        {t('重新檢查連線')}
                    </button>
                )}
            </div>
            {onboarding && <Feedback message={t(error)} />}
        </form>
    );
}
