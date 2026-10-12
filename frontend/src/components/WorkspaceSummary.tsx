import { useEffect, useState } from 'react';
import type { JSX } from 'react';

import { formatBytes } from '../lib/jobs.js';
import type { Status, Subscription } from '../lib/types.js';
import { Icon } from './Icon.js';
import { usePreferences } from './Preferences.js';
import { ProgressMeter } from './ProgressMeter.js';

function StorageMeter({ status }: { status: Status }): JSX.Element {
    const { language, t } = usePreferences();
    const used = Number(status.storage_used);
    const total = Number(status.storage_total);
    const valid =
        status.connected &&
        status.storage_used?.trim() &&
        status.storage_total?.trim() &&
        Number.isFinite(used) &&
        Number.isFinite(total) &&
        used >= 0 &&
        total > 0;
    const percent = valid ? Math.min(100, (used / total) * 100) : 0;
    const capacity = `${formatBytes(status.storage_used)} / ${formatBytes(status.storage_total)}`;
    const free = t(`剩餘 ${formatBytes(String(Math.max(0, total - used)))}`);
    const updated = status.storage_updated_at
        ? t(
              `上次更新 ${new Date(status.storage_updated_at * 1000).toLocaleTimeString(language === 'en' ? 'en-US' : 'zh-TW', { hour: '2-digit', minute: '2-digit', hour12: false })}`
          )
        : '';
    return (
        <section
            aria-label={t('PikPak 空間')}
            className='workspace-stat storage-stat'
            title={
                valid ? [free, updated].filter(Boolean).join(' · ') : undefined
            }
        >
            <div className='workspace-stat-text'>
                <Icon name='storage' />
                <span>{valid ? capacity : t('空間資料暫不可用')}</span>
                {valid && (
                    <span className='storage-percent'>
                        {Math.round(percent)}%
                    </span>
                )}
            </div>
            {valid && (
                <ProgressMeter
                    kind='meter'
                    label={t('PikPak 空間')}
                    segmented
                    value={percent}
                    valueText={`${capacity} · ${free}`}
                    warning={percent >= 90}
                />
            )}
        </section>
    );
}

function RssSchedule({
    subscriptions,
    status,
}: {
    subscriptions: Subscription[];
    status: Status;
}): JSX.Element {
    const { t } = usePreferences();
    const [now, setNow] = useState(() => Date.now() / 1000);
    useEffect(() => {
        const tick = () => {
            if (!document.hidden) {
                setNow(Date.now() / 1000);
            }
        };
        const timer = globalThis.setInterval(tick, 15_000);
        document.addEventListener('visibilitychange', tick);
        return () => {
            clearInterval(timer);
            document.removeEventListener('visibilitychange', tick);
        };
    }, []);
    const next = subscriptions
        .filter((sub) => sub.enabled)
        .toSorted((a, b) => (a.next_check || 0) - (b.next_check || 0))
        .at(0);
    const seconds = Math.max(0, (next?.next_check ?? 0) - now);
    const active = Boolean(next && status.connected && !status.paused);
    const label =
        !status.connected || status.paused
            ? t('RSS 已暫停')
            : next
              ? seconds <= 0
                  ? t('即將檢查')
                  : seconds <= 60
                    ? t(`${Math.ceil(seconds)} 秒後`)
                    : t(`${Math.ceil(seconds / 60)} 分鐘後`)
              : t('尚無啟用訂閱');
    const elapsed = next
        ? 100 * (1 - seconds / Math.max(1, next.interval_minutes * 60))
        : 0;
    return (
        <section
            aria-label={t('下次 RSS 檢查')}
            className='workspace-stat rss-schedule'
            title={next?.name}
        >
            <div className='workspace-stat-text'>
                <Icon name='rss' />
                <span>
                    {t('下次檢查')} · {label}
                </span>
            </div>
            {active && (
                <ProgressMeter
                    kind='meter'
                    label={t('RSS 檢查週期')}
                    value={elapsed}
                    valueText={label}
                />
            )}
        </section>
    );
}

export function WorkspaceSummary({
    status,
    subscriptions,
}: {
    status: Status;
    subscriptions: Subscription[];
}): JSX.Element {
    return (
        <div className='workspace-summary'>
            <RssSchedule status={status} subscriptions={subscriptions} />
            <StorageMeter status={status} />
        </div>
    );
}
