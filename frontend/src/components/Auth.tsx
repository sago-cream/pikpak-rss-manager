import { useState } from 'react';
import type { JSX } from 'react';

import { api } from '../lib/api.js';
import { errorMessage } from '../lib/i18n.js';
import type { Page } from '../lib/types.js';
import { CheckboxField, Feedback, FormField } from './FormField.js';
import { Appearance, usePreferences } from './Preferences.js';
import { TokenForm } from './TokenForm.js';
import { Wordmark } from './Wordmark.js';

export function Auth({
    page,
    version,
    onAuthenticated,
}: {
    page: Exclude<Page, 'app'>;
    version: string;
    onAuthenticated: () => Promise<void>;
}): JSX.Element {
    const { t } = usePreferences();
    const [code, setCode] = useState('');
    const [confirm, setConfirm] = useState('');
    const [error, setError] = useState('');
    const [password, setPassword] = useState('');
    const [pending, setPending] = useState(false);
    const [privateFeeds, setPrivateFeeds] = useState(false);
    const [publicURL, setPublicURL] = useState(
        import.meta.env.DEV
            ? import.meta.env.VITE_BACKEND_URL || 'http://127.0.0.1:8080'
            : globalThis.location.origin
    );
    const [recovered, setRecovered] = useState(false);
    const onboarding = page === 'onboarding';
    const recovery = page === 'recovery';
    const setup = page === 'setup';
    const title = recovered
        ? '密碼已重設'
        : setup
          ? '首次設定'
          : onboarding
            ? '連接 PikPak'
            : recovery
              ? '忘記密碼'
              : '登入';
    const submit = async () => {
        setError('');
        setPending(true);
        try {
            if ((setup || recovery) && password !== confirm) {
                throw new Error('兩次輸入的密碼不一致');
            }
            if (setup) {
                await api('/api/setup', 'POST', {
                    password,
                    confirm_password: confirm,
                    public_url: publicURL.trim(),
                    allow_private_feeds: privateFeeds,
                });
                await onAuthenticated();
            } else if (recovery) {
                await api('/api/password/recovery', 'POST', {
                    code: code.trim(),
                    new_password: password,
                    confirm_password: confirm,
                });
                setRecovered(true);
            } else {
                await api('/api/login', 'POST', { password });
                await onAuthenticated();
            }
        } catch (error_) {
            setError(errorMessage(error_));
        } finally {
            setPassword('');
            setConfirm('');
            setCode('');
            setPending(false);
        }
    };
    return (
        <div
            className={`login-page${setup || onboarding ? ' setup-page' : ''}`}
        >
            <main className='login-shell'>
                <div className='login-header'>
                    <Wordmark />
                    <Appearance
                        toggleLanguage={setup || onboarding || recovery}
                    />
                </div>
                {(setup || onboarding) && (
                    <ol
                        aria-label={t('設定進度')}
                        className='onboarding-progress'
                    >
                        <li aria-current={setup ? 'step' : undefined}>
                            <span className='step-number'>1</span>
                            <span>{t('建立管理員')}</span>
                        </li>
                        <li aria-current={onboarding ? 'step' : undefined}>
                            <span className='step-number'>2</span>
                            <span>{t('連接 PikPak')}</span>
                        </li>
                    </ol>
                )}
                <h1 id='recovery-title'>{t(title)}</h1>
                {recovery && !recovered && (
                    <div className='recovery-help' id='recovery-help'>
                        <p className='field-help'>
                            {t('在伺服器取得復原碼：')}
                        </p>
                        <code className='recovery-command' translate='no'>
                            docker compose exec pikpak-rss-manager
                            /app/pikpak-rss-manager recovery-code
                        </code>
                        <p className='field-help'>
                            {t('復原碼 15 分鐘內有效，只能使用一次。')}
                        </p>
                    </div>
                )}
                {onboarding ? (
                    <>
                        <TokenForm onboarding onSaved={onAuthenticated} />
                        <button
                            className='text-button onboarding-logout'
                            disabled={pending}
                            id='logout'
                            onClick={() => {
                                setPending(true);
                                void api('/api/logout', 'POST', {})
                                    .then(onAuthenticated)
                                    .catch((error_: unknown) => {
                                        setError(errorMessage(error_));
                                    })
                                    .finally(() => {
                                        setPending(false);
                                    });
                            }}
                            type='button'
                        >
                            {t('登出')}
                        </button>
                        <Feedback message={t(error)} />
                    </>
                ) : (
                    !recovered && (
                        <form
                            className='login-form form-stack'
                            id={
                                setup
                                    ? 'setup-form'
                                    : recovery
                                      ? 'recovery-form'
                                      : 'login-form'
                            }
                            onSubmit={(event) => {
                                event.preventDefault();
                                if (!pending) {
                                    void submit();
                                }
                            }}
                        >
                            {recovery && (
                                <FormField
                                    id='recovery-code'
                                    label={t('復原碼')}
                                >
                                    <input
                                        autoComplete='off'
                                        onChange={(event) => {
                                            setCode(event.target.value);
                                        }}
                                        required
                                        type='password'
                                        value={code}
                                    />
                                </FormField>
                            )}
                            <FormField
                                id={
                                    setup
                                        ? 'setup-password'
                                        : recovery
                                          ? 'recovery-password'
                                          : 'password'
                                }
                                label={t(recovery ? '新密碼' : '密碼')}
                            >
                                <input
                                    autoComplete={
                                        setup || recovery
                                            ? 'new-password'
                                            : 'current-password'
                                    }
                                    onChange={(event) => {
                                        setPassword(event.target.value);
                                    }}
                                    placeholder={t(
                                        setup
                                            ? '密碼'
                                            : recovery
                                              ? ''
                                              : '輸入你的密碼'
                                    )}
                                    required
                                    type='password'
                                    value={password}
                                />
                            </FormField>
                            {(setup || recovery) && (
                                <FormField
                                    id={
                                        setup
                                            ? 'setup-confirm'
                                            : 'recovery-confirm'
                                    }
                                    label={t(
                                        recovery
                                            ? '再次輸入新密碼'
                                            : '再次輸入密碼'
                                    )}
                                >
                                    <input
                                        autoComplete='new-password'
                                        onChange={(event) => {
                                            setConfirm(event.target.value);
                                        }}
                                        placeholder={t(setup ? '相同密碼' : '')}
                                        required
                                        type='password'
                                        value={confirm}
                                    />
                                </FormField>
                            )}
                            {setup && (
                                <>
                                    <FormField
                                        id='setup-public-url'
                                        info={{
                                            id: 'setup-url-help',
                                            label: t('網站網址說明'),
                                            content: t(
                                                '預填目前網址。使用反向代理時填入對外網址。'
                                            ),
                                        }}
                                        label={t('網站網址')}
                                    >
                                        <input
                                            onChange={(event) => {
                                                setPublicURL(
                                                    event.target.value
                                                );
                                            }}
                                            placeholder='https://rss.example.com'
                                            type='url'
                                            value={publicURL}
                                        />
                                    </FormField>
                                    <CheckboxField
                                        checked={privateFeeds}
                                        id='setup-private-feeds'
                                        info={{
                                            id: 'setup-private-help',
                                            label: t('內網 RSS 說明'),
                                            content:
                                                t(
                                                    '僅在訂閱來源位於內網時啟用。'
                                                ),
                                        }}
                                        label={t('允許存取內網 RSS')}
                                        onChange={setPrivateFeeds}
                                    />
                                </>
                            )}
                            <button
                                className='button primary'
                                disabled={pending}
                                type='submit'
                            >
                                {t(
                                    setup
                                        ? '下一步'
                                        : recovery
                                          ? '重設密碼'
                                          : '登入'
                                )}
                            </button>
                            <Feedback message={t(error)} />
                        </form>
                    )
                )}
                {recovered && (
                    <p
                        className='field-help'
                        id='recovery-success'
                        role='status'
                    >
                        {t('請使用新密碼登入。')}
                    </p>
                )}
                {recovery ? (
                    <a
                        className='text-link recovery-back'
                        href={import.meta.env.DEV ? '/ui/' : '/'}
                    >
                        {t('返回登入')}
                    </a>
                ) : (
                    !setup &&
                    !onboarding && (
                        <>
                            <a
                                className='text-link forgot-password'
                                href={
                                    import.meta.env.DEV
                                        ? '/ui/forgot-password'
                                        : '/forgot-password'
                                }
                            >
                                {t('忘記密碼？')}
                            </a>
                            <div className='login-footer'>
                                <span className='version'>{version}</span>
                            </div>
                        </>
                    )
                )}
            </main>
        </div>
    );
}
