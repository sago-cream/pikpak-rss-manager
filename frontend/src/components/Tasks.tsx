import { useState } from 'react';
import type { JSX } from 'react';

import { filterJobs } from '../lib/jobs.js';
import type { JobFilter } from '../lib/jobs.js';
import type { Job } from '../lib/types.js';
import { AddButton } from './AddButton.js';
import { Badge } from './Badge.js';
import { Icon } from './Icon.js';
import { usePreferences } from './Preferences.js';
import { Select } from './Select.js';

export function Tasks({
    jobs,
    date,
    onOpen,
    onAdd,
    onClear,
    busy,
}: {
    jobs: Job[];
    date: (value: number) => string;
    onOpen: (id: string) => void;
    onAdd: () => void;
    onClear: () => void;
    busy: boolean;
}): JSX.Element {
    const { t } = usePreferences();
    const [filter, setFilter] = useState<JobFilter>('all');
    const [limit, setLimit] = useState(8);
    const items = filterJobs(jobs, filter);
    return (
        <section
            aria-labelledby='jobs-title'
            className='workspace-section'
            id='view-jobs'
        >
            <div className='workspace-section-heading'>
                <h2 id='jobs-title'>{t('離線任務')}</h2>
                <div className='section-actions'>
                    {filter === 'complete' && items.length > 0 && (
                        <button
                            className='text-button'
                            disabled={busy}
                            id='clear-completed-jobs'
                            onClick={onClear}
                            type='button'
                        >
                            {t('清除已完成任務')}
                        </button>
                    )}
                    <AddButton kind='task' onClick={onAdd} />
                </div>
            </div>
            <div id='jobs-list'>
                <div className='table-scroll'>
                    <table>
                        <thead>
                            <tr>
                                <th scope='col'>{t('發布與作品')}</th>
                                <th className='job-status-column' scope='col'>
                                    <Select
                                        aria-label={t('任務狀態')}
                                        className='job-status-filter'
                                        id='job-status-filter'
                                        onChange={(event) => {
                                            setFilter(
                                                event.target.value as JobFilter
                                            );
                                            setLimit(8);
                                        }}
                                        value={filter}
                                    >
                                        {Object.entries({
                                            all: '全部',
                                            running: '處理中',
                                            review: '待處理',
                                            complete: '已完成',
                                        }).map(([value, label]) => (
                                            <option key={value} value={value}>
                                                {t('狀態')} · {t(label)}
                                            </option>
                                        ))}
                                    </Select>
                                </th>
                                <th scope='col'>{t('時間')}</th>
                                <th scope='col'>
                                    <span className='sr-only'>{t('查看')}</span>
                                </th>
                            </tr>
                        </thead>
                        <tbody>
                            {items.slice(0, limit).map((job) => (
                                <tr key={job.id}>
                                    <td className='job-name'>
                                        <strong>{job.rule.title}</strong>
                                        <span title={job.title}>
                                            {job.title}
                                        </span>
                                        {job.error && (
                                            <small className='error'>
                                                {t(job.error)}
                                            </small>
                                        )}
                                    </td>
                                    <td>
                                        <Badge state={job.state} />
                                        {job.state === 'downloading' && (
                                            <div className='progress-caption'>
                                                {Math.round(job.progress)}%
                                            </div>
                                        )}
                                    </td>
                                    <td className='muted nowrap'>
                                        {date(job.created_at)}
                                    </td>
                                    <td>
                                        <div className='job-actions'>
                                            <button
                                                className='text-button'
                                                data-job={job.id}
                                                onClick={() => {
                                                    onOpen(job.id);
                                                }}
                                                type='button'
                                            >
                                                {t('查看')}
                                                <Icon name='arrow-right' />
                                            </button>
                                        </div>
                                    </td>
                                </tr>
                            ))}
                            {items.length === 0 && (
                                <tr className='jobs-empty'>
                                    <td colSpan={4}>
                                        <p className='workspace-idle'>
                                            {t(
                                                jobs.length > 0
                                                    ? '沒有符合的任務'
                                                    : '尚無任務'
                                            )}
                                        </p>
                                    </td>
                                </tr>
                            )}
                        </tbody>
                    </table>
                </div>
            </div>
            {items.length > limit && (
                <button
                    className='text-button'
                    id='jobs-more'
                    onClick={() => {
                        setLimit((value) => value + 8);
                    }}
                    type='button'
                >
                    {t('顯示更多')}
                </button>
            )}
        </section>
    );
}
