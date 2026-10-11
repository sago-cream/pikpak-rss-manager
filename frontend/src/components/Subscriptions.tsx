import type { JSX } from 'react';

import type { Subscription } from '../lib/types.js';
import { AddButton } from './AddButton.js';
import { Icon } from './Icon.js';
import { usePreferences } from './Preferences.js';

export function Subscriptions({
    subscriptions,
    date,
    busy,
    onEdit,
    onAdd,
    onToggle,
}: {
    subscriptions: Subscription[];
    date: (value: number) => string;
    busy: boolean;
    onEdit: (sub: Subscription) => void;
    onAdd: () => void;
    onToggle: (sub: Subscription) => void;
}): JSX.Element {
    const { t } = usePreferences();
    return (
        <section
            aria-labelledby='subscriptions-title'
            className='workspace-section'
            id='view-subscriptions'
        >
            <div className='workspace-section-heading'>
                <h2 id='subscriptions-title'>{t('訂閱管理')}</h2>
                <AddButton kind='subscription' onClick={onAdd} />
            </div>
            <div id='subscriptions-list'>
                {subscriptions.map((sub) => (
                    <article
                        className='subscription-row'
                        data-paused={!sub.enabled}
                        key={sub.id}
                    >
                        <button
                            aria-label={`${t(sub.enabled ? '暫停追蹤' : '繼續追蹤')} ${sub.name}`}
                            className='icon-button subscription-toggle'
                            data-toggle={sub.id}
                            disabled={busy}
                            onClick={() => {
                                onToggle(sub);
                            }}
                            title={t(sub.enabled ? '暫停追蹤' : '繼續追蹤')}
                            type='button'
                        >
                            <Icon name={sub.enabled ? 'pause' : 'play'} />
                        </button>
                        <div className='subscription-info'>
                            <h3>
                                <button
                                    aria-label={`${t('訂閱設定')} ${sub.name}`}
                                    className='subscription-open'
                                    data-edit={sub.id}
                                    onClick={() => {
                                        onEdit(sub);
                                    }}
                                    type='button'
                                >
                                    {sub.name}
                                </button>
                            </h3>
                            <div className='subscription-meta'>
                                <span
                                    className='destination-text'
                                    title={sub.destination || t('根目錄')}
                                >
                                    {sub.destination || t('根目錄')}
                                </span>
                                <span className='subscription-schedule'>
                                    <time title={t('上次檢查')}>
                                        {date(sub.last_checked)}
                                    </time>
                                    {' · '}
                                    {t(`每 ${sub.interval_minutes} 分鐘`)}
                                </span>
                            </div>
                            {sub.last_error && (
                                <p className='error'>{t(sub.last_error)}</p>
                            )}
                        </div>
                        <button
                            aria-label={`${t('訂閱設定')} ${sub.name}`}
                            className='icon-button subscription-menu'
                            data-subscription-settings={sub.id}
                            onClick={() => {
                                onEdit(sub);
                            }}
                            title={t('訂閱設定')}
                            type='button'
                        >
                            <Icon name='more-vertical' />
                        </button>
                    </article>
                ))}
                {subscriptions.length === 0 && (
                    <p className='workspace-idle'>{t('尚無訂閱')}</p>
                )}
            </div>
        </section>
    );
}
