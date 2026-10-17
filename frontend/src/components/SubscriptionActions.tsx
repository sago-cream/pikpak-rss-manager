import { useState } from 'react';
import type { JSX } from 'react';

import { api } from '../lib/api.js';
import { errorMessage } from '../lib/i18n.js';
import { useLifetime } from '../lib/lifetime.js';
import type { Subscription } from '../lib/types.js';
import { BackfillPanel } from './BackfillPanel.js';
import { Feedback } from './FormField.js';
import { Icon } from './Icon.js';
import { usePreferences } from './Preferences.js';

export function SubscriptionActions({
    subscription,
    onSaved,
    onDeleted,
}: {
    subscription: Subscription;
    onSaved: () => Promise<void>;
    onDeleted: () => void;
}): JSX.Element {
    const { t } = usePreferences();
    const active = useLifetime();
    const [backfill, setBackfill] = useState(false);
    const [busy, setBusy] = useState('');
    const [error, setError] = useState('');
    const [checked, setChecked] = useState(false);
    const run = async (remove: boolean) => {
        if (
            busy ||
            (remove &&
                !globalThis.confirm(
                    t('刪除這筆訂閱？已建立的任務與雲端檔案會保留。')
                ))
        ) {
            return;
        }
        setBusy(remove ? 'delete' : 'check');
        setError('');
        setChecked(false);
        try {
            await api(
                `/api/subscriptions/${subscription.id}${remove ? '' : '/check'}`,
                remove ? 'DELETE' : 'POST',
                remove ? undefined : { backfill: false }
            );
            await onSaved();
            if (active.current) {
                if (remove) {
                    onDeleted();
                } else {
                    setChecked(true);
                }
            }
        } catch (error_) {
            if (active.current) {
                setError(errorMessage(error_));
            }
        } finally {
            if (active.current) {
                setBusy('');
            }
        }
    };
    return (
        <div className='subscription-management form-stack'>
            <h2>{t('管理訂閱')}</h2>
            <p className='field-help'>{t('管理操作使用已儲存的設定。')}</p>
            <Feedback
                message={t(error || (checked ? '' : subscription.last_error))}
            />
            {backfill ? (
                <BackfillPanel
                    onClose={() => {
                        setBackfill(false);
                    }}
                    onSaved={onSaved}
                    subscriptionID={subscription.id}
                />
            ) : (
                <>
                    <div className='form-actions'>
                        <button
                            aria-busy={busy === 'check'}
                            className='button subtle'
                            data-check={subscription.id}
                            disabled={Boolean(busy)}
                            onClick={() => {
                                void run(false);
                            }}
                            type='button'
                        >
                            <Icon
                                className={
                                    busy === 'check' ? 'is-loading' : undefined
                                }
                                name={busy === 'check' ? 'loading' : 'check'}
                            />
                            {t('檢查新的發布')}
                        </button>
                        <button
                            className='button subtle'
                            data-backfill={subscription.id}
                            disabled={Boolean(busy)}
                            onClick={() => {
                                setBackfill(true);
                            }}
                            type='button'
                        >
                            <Icon name='backfill' />
                            {t('補抓')}
                        </button>
                    </div>
                    {checked && <p role='status'>{t('訂閱檢查完成')}</p>}
                    <div className='subscription-delete'>
                        <button
                            aria-busy={busy === 'delete'}
                            className='button danger'
                            data-delete={subscription.id}
                            disabled={Boolean(busy)}
                            onClick={() => {
                                void run(true);
                            }}
                            type='button'
                        >
                            <Icon
                                className={
                                    busy === 'delete' ? 'is-loading' : undefined
                                }
                                name={busy === 'delete' ? 'loading' : 'trash'}
                            />
                            {t('刪除訂閱')}
                        </button>
                    </div>
                </>
            )}
        </div>
    );
}
