import type { JSX } from 'react';

import type { Subscription } from '../lib/types.js';
import { AddButton } from './AddButton.js';
import { Icon } from './Icon.js';
import { usePreferences } from './Preferences.js';
import { RowMetadata } from './RowMetadata.js';
import { WorkspacePanel } from './WorkspacePanel.js';
import { WorkspaceRow } from './WorkspaceRow.js';

export function Subscriptions({
    subscriptions,
    date,
    pending,
    onEdit,
    onAdd,
    onToggle,
}: {
    subscriptions: Subscription[];
    date: (value: number) => string;
    pending: ReadonlySet<string>;
    onEdit: (sub: Subscription) => void;
    onAdd: () => void;
    onToggle: (sub: Subscription) => void;
}): JSX.Element {
    const { t } = usePreferences();
    return (
        <WorkspacePanel
            actions={<AddButton kind='subscription' onClick={onAdd} />}
            id='view-subscriptions'
            listId='subscriptions-list'
            listLabel={t('訂閱清單')}
            title={t('RSS 訂閱')}
            titleId='subscriptions-title'
        >
            {subscriptions.map((sub) => (
                <WorkspaceRow
                    disabled={pending.has(`toggle-${sub.id}`)}
                    id={sub.id}
                    key={sub.id}
                    kind='subscription'
                    label={`${t('訂閱設定')} ${sub.name}`}
                    leading={
                        <span className='subscription-indicator'>
                            <Icon name='file-symlink' />
                        </span>
                    }
                    onOpen={() => {
                        onEdit(sub);
                    }}
                    paused={!sub.enabled}
                    title={sub.name}
                    trailing={
                        <button
                            aria-busy={pending.has(`toggle-${sub.id}`)}
                            aria-label={`${t(sub.enabled ? '暫停追蹤' : '繼續追蹤')} ${sub.name}`}
                            className='icon-button subscription-toggle'
                            data-toggle={sub.id}
                            disabled={pending.has(`toggle-${sub.id}`)}
                            onClick={() => {
                                onToggle(sub);
                            }}
                            title={t(sub.enabled ? '暫停追蹤' : '繼續追蹤')}
                            type='button'
                        >
                            <Icon
                                className={
                                    pending.has(`toggle-${sub.id}`)
                                        ? 'is-loading'
                                        : undefined
                                }
                                name={
                                    pending.has(`toggle-${sub.id}`)
                                        ? 'loading'
                                        : sub.enabled
                                          ? 'pause'
                                          : 'play'
                                }
                            />
                        </button>
                    }
                >
                    <RowMetadata
                        items={[
                            <span title={sub.destination || t('根目錄')}>
                                {sub.destination || t('根目錄')}
                            </span>,
                            <time title={t('上次檢查')}>
                                {date(sub.last_checked)}
                            </time>,
                            t(`每 ${sub.interval_minutes} 分鐘`),
                        ]}
                    />
                    {sub.last_error && (
                        <p className='error'>{t(sub.last_error)}</p>
                    )}
                </WorkspaceRow>
            ))}
            {subscriptions.length === 0 && (
                <p className='workspace-idle'>{t('尚無訂閱')}</p>
            )}
        </WorkspacePanel>
    );
}
