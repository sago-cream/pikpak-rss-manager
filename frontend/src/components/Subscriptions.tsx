import { useState } from 'react';
import type { JSX } from 'react';

import type { Subscription } from '../lib/types.js';
import { Icon } from './Icon.js';
import { usePreferences } from './Preferences.js';

export function Subscriptions({
    subscriptions,
    date,
    busy,
    onEdit,
    onBackfill,
    onCheck,
    onDelete,
}: {
    subscriptions: Subscription[];
    date: (value: number) => string;
    busy: boolean;
    onEdit: (sub: Subscription) => void;
    onBackfill: (id: number) => void;
    onCheck: (id: number) => void;
    onDelete: (id: number) => void;
}): JSX.Element {
    const { t } = usePreferences();
    const [search, setSearch] = useState('');
    const items = subscriptions.filter((sub) =>
        sub.name.toLowerCase().includes(search.toLowerCase())
    );
    return (
        <section
            aria-labelledby='subscriptions-title'
            className='workspace-section'
            id='view-subscriptions'
        >
            <div className='workspace-section-heading'>
                <h2 id='subscriptions-title'>{t('訂閱管理')}</h2>
            </div>
            <div className='toolbar'>
                <input
                    aria-label={t('搜尋訂閱')}
                    id='subscription-search'
                    onChange={(event) => {
                        setSearch(event.target.value);
                    }}
                    placeholder={t('搜尋作品或訂閱名稱')}
                    type='search'
                    value={search}
                />
            </div>
            <div id='subscriptions-list'>
                {items.map((sub) => (
                    <article className='subscription-row' key={sub.id}>
                        <div className='subscription-summary'>
                            <span className='subscription-symbol'>
                                <Icon name='rss' />
                            </span>
                            <div className='subscription-info'>
                                <div className='subscription-name'>
                                    <h3>{sub.name}</h3>
                                    <span
                                        className={`badge ${sub.enabled ? 'green' : ''}`}
                                    >
                                        {t(sub.enabled ? '追蹤中' : '已停用')}
                                    </span>
                                </div>
                                <div className='subscription-meta'>
                                    <span
                                        title={sub.destination || t('根目錄')}
                                    >
                                        <Icon name='folder' />
                                        <span className='destination-text'>
                                            {sub.destination || t('根目錄')}
                                        </span>
                                    </span>
                                    <span>
                                        <Icon name='clock' />
                                        {t(`每 ${sub.interval_minutes} 分鐘`)}
                                    </span>
                                </div>
                            </div>
                        </div>
                        {sub.last_error && (
                            <p className='error'>{t(sub.last_error)}</p>
                        )}
                        <div className='subscription-footer'>
                            <time className='muted' title={t('上次檢查')}>
                                {date(sub.last_checked)}
                            </time>
                            <div className='subscription-actions'>
                                <button
                                    aria-label={t('編輯訂閱')}
                                    className='icon-button'
                                    data-edit={sub.id}
                                    onClick={() => {
                                        onEdit(sub);
                                    }}
                                    title={t('編輯訂閱')}
                                    type='button'
                                >
                                    <Icon name='edit' />
                                </button>
                                <button
                                    aria-label={t('檢查新的發布')}
                                    className='icon-button'
                                    data-check={sub.id}
                                    disabled={busy}
                                    onClick={() => {
                                        onCheck(sub.id);
                                    }}
                                    title={t('檢查新的發布')}
                                    type='button'
                                >
                                    <Icon name='check' />
                                </button>
                                <button
                                    aria-label={t('補抓')}
                                    className='icon-button'
                                    data-backfill={sub.id}
                                    onClick={() => {
                                        onBackfill(sub.id);
                                    }}
                                    title={t('處理 RSS 中仍可取得的現有項目')}
                                    type='button'
                                >
                                    <Icon name='backfill' />
                                </button>
                                <button
                                    aria-label={`${t('刪除訂閱')} ${sub.name}`}
                                    className='icon-button danger'
                                    data-delete={sub.id}
                                    disabled={busy}
                                    onClick={() => {
                                        onDelete(sub.id);
                                    }}
                                    title={t('刪除訂閱')}
                                    type='button'
                                >
                                    <Icon name='trash' />
                                </button>
                            </div>
                        </div>
                    </article>
                ))}
                {items.length === 0 && (
                    <p className='workspace-idle'>
                        {t(
                            subscriptions.length > 0 ? '找不到訂閱' : '尚無訂閱'
                        )}
                    </p>
                )}
            </div>
        </section>
    );
}
