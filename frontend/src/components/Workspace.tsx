import { useCallback, useEffect, useRef, useState } from 'react';
import type { JSX } from 'react';

import brand from '../assets/favicon.svg';
import { api } from '../lib/api.js';
import { errorMessage, isCancelled } from '../lib/i18n.js';
import type {
    Job,
    LogEntry,
    Status,
    Subscription,
    WorkspaceData,
} from '../lib/types.js';
import { Dialog } from './Dialog.js';
import { Icon } from './Icon.js';
import { JobDialog } from './JobDialog.js';
import { Appearance, usePreferences } from './Preferences.js';
import { Settings } from './Settings.js';
import { SubscriptionDialog } from './SubscriptionDialog.js';
import { Subscriptions } from './Subscriptions.js';
import { Tasks } from './Tasks.js';

type Modal =
    | { type: 'subscription'; subscription?: Subscription; manual?: boolean }
    | { type: 'job'; id: string }
    | { type: 'settings' | 'events' };
export function Workspace({
    version,
    onAuthenticated,
}: {
    version: string;
    onAuthenticated: () => Promise<void>;
}): JSX.Element {
    const { language, t } = usePreferences();
    const [busy, setBusy] = useState<ReadonlySet<string>>(new Set());
    const pending = useRef(new Set<string>());
    const [mobileView, setMobileView] = useState(
        globalThis.location.hash === '#jobs' ? 'jobs' : 'subscriptions'
    );
    const [data, setData] = useState<WorkspaceData | null>(null);
    const [feedback, setFeedback] = useState('');
    const [feedbackError, setFeedbackError] = useState(false);
    const [modal, setModal] = useState<Modal | null>(() => {
        const view = globalThis.location.hash.slice(1);
        return view === 'settings' || view === 'events' ? { type: view } : null;
    });
    const initialScroll = useRef(false);
    const profile = useRef<HTMLDivElement>(null);
    const profileButton = useRef<HTMLButtonElement>(null);
    const request = useRef<AbortController | null>(null);
    const refresh = useCallback(async () => {
        request.current?.abort();
        const controller = new AbortController();
        request.current = controller;
        try {
            const [subscriptions, jobs, events, status] = await Promise.all([
                api<Subscription[]>(
                    '/api/subscriptions',
                    'GET',
                    undefined,
                    controller.signal
                ),
                api<Job[]>('/api/jobs', 'GET', undefined, controller.signal),
                api<LogEntry[]>(
                    '/api/events',
                    'GET',
                    undefined,
                    controller.signal
                ),
                api<Status>(
                    '/api/settings/pikpak',
                    'GET',
                    undefined,
                    controller.signal
                ),
            ]);
            if (!controller.signal.aborted) {
                setData({ subscriptions, jobs, events, status });
            }
        } catch (error) {
            if (!controller.signal.aborted && !isCancelled(error)) {
                throw error;
            }
        }
    }, []);
    useEffect(() => {
        // The API response updates state asynchronously; cleanup cancels the request.
        // eslint-disable-next-line react-hooks/set-state-in-effect
        void refresh().catch((error: unknown) => {
            setFeedback(errorMessage(error));
            setFeedbackError(true);
        });
        return () => request.current?.abort();
    }, [refresh]);
    useEffect(() => {
        if (!data || initialScroll.current) {
            return;
        }
        initialScroll.current = true;
        const view = globalThis.location.hash.slice(1);
        if (view === 'jobs' || view === 'subscriptions') {
            document
                .querySelector(`#view-${view}`)
                ?.scrollIntoView({ block: 'start' });
        }
    }, [data]);
    useEffect(() => {
        if (modal) {
            return;
        }
        const timer = globalThis.setInterval(() => {
            if (!document.hidden) {
                void refresh().catch((error: unknown) => {
                    setFeedback(errorMessage(error));
                    setFeedbackError(true);
                });
            }
        }, 15_000);
        return () => {
            clearInterval(timer);
        };
    }, [modal, refresh]);
    useEffect(() => {
        if (!feedback) {
            return;
        }
        const timer = globalThis.setTimeout(() => {
            setFeedback('');
        }, 4500);
        return () => {
            clearTimeout(timer);
        };
    }, [feedback]);
    const action = async (
        key: string,
        operation: () => Promise<void>,
        message: string
    ) => {
        if (pending.current.has(key)) {
            return;
        }
        pending.current.add(key);
        setBusy(new Set(pending.current));
        try {
            await operation();
            setFeedback(message);
            setFeedbackError(false);
        } catch (error) {
            setFeedback(errorMessage(error));
            setFeedbackError(true);
        } finally {
            pending.current.delete(key);
            setBusy(new Set(pending.current));
        }
    };
    const date = (timestamp: number) =>
        timestamp
            ? new Date(timestamp * 1000).toLocaleString(
                  language === 'en' ? 'en-US' : 'zh-TW',
                  {
                      month: '2-digit',
                      day: '2-digit',
                      hour: '2-digit',
                      minute: '2-digit',
                      hour12: false,
                  }
              )
            : t('尚未檢查');
    const close = () => {
        setModal(null);
        globalThis.history.replaceState(null, '', '#overview');
    };
    const utility = (type: 'settings' | 'events') => {
        profile.current?.hidePopover();
        setModal({ type });
        globalThis.history.replaceState(null, '', `#${type}`);
    };
    const connected = data?.status.connected && !data.status.paused;
    return (
        <>
            <div className='app-shell'>
                <header className='workspace-header'>
                    <a
                        className='brand'
                        href={import.meta.env.DEV ? '/ui/' : '/'}
                    >
                        <img alt='' className='brand-mark small' src={brand} />
                        <span>
                            PikPak <b>RSS</b>
                        </span>
                    </a>
                    <button
                        aria-busy={busy.has('refresh')}
                        aria-label={t('重新整理')}
                        className='icon-button workspace-refresh'
                        disabled={busy.has('refresh')}
                        id='refresh'
                        onClick={() => {
                            void action('refresh', refresh, '已更新');
                        }}
                        title={t('重新整理')}
                        type='button'
                    >
                        <Icon
                            className={
                                busy.has('refresh') ? 'is-loading' : undefined
                            }
                            name={busy.has('refresh') ? 'loading' : 'refresh'}
                        />
                    </button>
                    <button
                        aria-label={t('帳號選單')}
                        className='profile-toggle'
                        id='profile-toggle'
                        popoverTarget='profile-menu'
                        ref={profileButton}
                        type='button'
                    >
                        <span aria-hidden='true' className='profile-avatar'>
                            P
                        </span>
                        <span id='profile-name'>
                            {data?.status.name || t('PikPak 帳號')}
                        </span>
                        <Icon name='chevron' />
                    </button>
                </header>
                <div
                    className='profile-menu'
                    id='profile-menu'
                    onBeforeToggle={(event) => {
                        if (
                            !profileButton.current ||
                            event.currentTarget.matches(':popover-open')
                        ) {
                            return;
                        }
                        const rect =
                            profileButton.current.getBoundingClientRect();
                        event.currentTarget.style.top = `${rect.bottom + 8}px`;
                        event.currentTarget.style.right = `${Math.max(16, window.innerWidth - rect.right)}px`;
                    }}
                    popover='auto'
                    ref={profile}
                >
                    <div className='connection-pill' id='profile-connection'>
                        <span
                            className={`dot ${connected ? 'green' : 'amber'}`}
                        />
                        {t(
                            connected
                                ? 'PikPak 已連線'
                                : data?.status.paused
                                  ? 'PikPak 已暫停'
                                  : '尚未綁定 PikPak'
                        )}
                    </div>
                    <button
                        className='profile-action'
                        data-view='settings'
                        onClick={() => {
                            utility('settings');
                        }}
                        type='button'
                    >
                        {t('系統設定')}
                    </button>
                    <button
                        className='profile-action'
                        data-view='events'
                        onClick={() => {
                            utility('events');
                        }}
                        type='button'
                    >
                        {t('執行日誌')}
                    </button>
                    <Appearance />
                    <div className='profile-footer'>
                        <span className='version'>{version}</span>
                        <button
                            aria-busy={busy.has('logout')}
                            className='text-button'
                            disabled={busy.has('logout')}
                            id='logout'
                            onClick={() => {
                                void action(
                                    'logout',
                                    async () => {
                                        await api('/api/logout', 'POST', {});
                                        await onAuthenticated();
                                    },
                                    ''
                                );
                            }}
                            type='button'
                        >
                            {t('登出')}
                        </button>
                    </div>
                </div>
                <main aria-label={t('總覽')} className='main'>
                    <div className='content' id='view-overview'>
                        <div id='overview-content'>
                            {data?.status.paused && (
                                <div className='overview-connection'>
                                    <span>{t('PikPak 已暫停')}</span>
                                    <button
                                        className='text-button'
                                        data-view='settings'
                                        onClick={() => {
                                            utility('settings');
                                        }}
                                        type='button'
                                    >
                                        {t('修復連線')}
                                        <Icon name='arrow-right' />
                                    </button>
                                </div>
                            )}
                        </div>
                        {data && (
                            <>
                                <nav
                                    aria-label={t('工作區')}
                                    className='workspace-switch'
                                >
                                    {(['subscriptions', 'jobs'] as const).map(
                                        (view) => (
                                            <button
                                                aria-controls={`view-${view}`}
                                                aria-pressed={
                                                    mobileView === view
                                                }
                                                data-workspace-tab={view}
                                                key={view}
                                                onClick={() => {
                                                    setMobileView(view);
                                                }}
                                                type='button'
                                            >
                                                {t(
                                                    view === 'subscriptions'
                                                        ? '訂閱'
                                                        : '任務'
                                                )}
                                            </button>
                                        )
                                    )}
                                </nav>
                                <div
                                    className='workspace-body'
                                    data-mobile-view={mobileView}
                                >
                                    <Subscriptions
                                        date={date}
                                        onAdd={() => {
                                            setModal({ type: 'subscription' });
                                        }}
                                        onEdit={(subscription) => {
                                            setModal({
                                                type: 'subscription',
                                                subscription,
                                            });
                                        }}
                                        onToggle={(subscription) => {
                                            void action(
                                                `toggle-${subscription.id}`,
                                                async () => {
                                                    await api(
                                                        `/api/subscriptions/${subscription.id}`,
                                                        'PUT',
                                                        {
                                                            name: subscription.name,
                                                            rss_url:
                                                                subscription.rss_url,
                                                            destination:
                                                                subscription.destination,
                                                            destination_id:
                                                                subscription.destination_id ||
                                                                '',
                                                            destination_account_ref:
                                                                subscription.destination_account_ref ||
                                                                '',
                                                            interval_minutes:
                                                                subscription.interval_minutes,
                                                            rename_enabled:
                                                                subscription.rename_enabled,
                                                            regex: subscription.regex,
                                                            replacement:
                                                                subscription.replacement,
                                                            enabled:
                                                                !subscription.enabled,
                                                        }
                                                    );
                                                    setData((current) =>
                                                        current
                                                            ? {
                                                                  ...current,
                                                                  subscriptions:
                                                                      current.subscriptions.map(
                                                                          (
                                                                              sub
                                                                          ) =>
                                                                              sub.id ===
                                                                              subscription.id
                                                                                  ? {
                                                                                        ...sub,
                                                                                        enabled:
                                                                                            !subscription.enabled,
                                                                                    }
                                                                                  : sub
                                                                      ),
                                                              }
                                                            : current
                                                    );
                                                    await refresh();
                                                },
                                                subscription.enabled
                                                    ? '已暫停追蹤'
                                                    : '已繼續追蹤'
                                            );
                                        }}
                                        pending={busy}
                                        subscriptions={data.subscriptions}
                                    />
                                    <Tasks
                                        busy={busy.has('clear')}
                                        date={date}
                                        jobs={data.jobs}
                                        onAdd={() => {
                                            setModal({
                                                type: 'subscription',
                                                manual: true,
                                            });
                                        }}
                                        onClear={() => {
                                            if (
                                                globalThis.confirm(
                                                    t(
                                                        '清除所有已完成任務紀錄？雲端檔案會保留。'
                                                    )
                                                )
                                            ) {
                                                void action(
                                                    'clear',
                                                    async () => {
                                                        await api(
                                                            '/api/jobs/completed',
                                                            'DELETE'
                                                        );
                                                        await refresh();
                                                    },
                                                    '已清除已完成任務'
                                                );
                                            }
                                        }}
                                        onOpen={(id) => {
                                            setModal({ type: 'job', id });
                                        }}
                                    />
                                </div>
                            </>
                        )}
                    </div>
                </main>
            </div>
            {modal?.type === 'subscription' && data && (
                <SubscriptionDialog
                    key={
                        modal.subscription?.id ??
                        (modal.manual ? 'manual' : 'new')
                    }
                    manual={modal.manual}
                    onClose={close}
                    onSaved={refresh}
                    status={data.status}
                    subscription={modal.subscription}
                />
            )}
            {modal?.type === 'job' && (
                <JobDialog
                    connected={Boolean(connected)}
                    jobID={modal.id}
                    onChanged={refresh}
                    onClose={close}
                    onFixConnection={() => {
                        utility('settings');
                    }}
                />
            )}
            {modal &&
                (modal.type === 'settings' || modal.type === 'events') &&
                data && (
                    <Dialog
                        id='utility-dialog'
                        onClose={close}
                        title={t(
                            modal.type === 'settings' ? '系統設定' : '執行日誌'
                        )}
                    >
                        {modal.type === 'settings' ? (
                            <Settings
                                onAuthenticated={onAuthenticated}
                                onRefresh={refresh}
                                status={data.status}
                            />
                        ) : (
                            <section className='utility-view' id='view-events'>
                                <div className='panel'>
                                    <div className='panel-heading'>
                                        <h2>{t('執行紀錄')}</h2>
                                        <span className='muted'>
                                            {t('保留 30 天，顯示最近 150 筆')}
                                        </span>
                                    </div>
                                    <div id='events-list'>
                                        {data.events.map((event) => (
                                            <div
                                                className='event-row'
                                                key={event.id}
                                            >
                                                <span
                                                    className={`event-dot ${event.level}`}
                                                />
                                                <div>
                                                    <p>{t(event.message)}</p>
                                                    <small className='muted'>
                                                        {event.job_id
                                                            ? `${t('任務')} ${event.job_id.slice(0, 8)}`
                                                            : event.subscription_id
                                                              ? `${t('訂閱')} #${event.subscription_id}`
                                                              : t('系統')}
                                                    </small>
                                                </div>
                                                <time>
                                                    {date(event.created_at)}
                                                </time>
                                            </div>
                                        ))}
                                        {data.events.length === 0 && (
                                            <div className='empty-state'>
                                                <h3>{t('尚無日誌')}</h3>
                                            </div>
                                        )}
                                    </div>
                                </div>
                            </section>
                        )}
                    </Dialog>
                )}
            {feedback && (
                <div
                    className={`toast${feedbackError ? ' is-error' : ''}`}
                    id='toast'
                    role='status'
                >
                    {t(feedback)}
                </div>
            )}
        </>
    );
}
