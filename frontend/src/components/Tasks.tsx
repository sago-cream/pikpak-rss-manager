import { useState } from 'react';
import type { JSX } from 'react';

import { jobColor, jobLabels, sortJobs } from '../lib/jobs.js';
import type { Job } from '../lib/types.js';
import { AddButton } from './AddButton.js';
import { Icon } from './Icon.js';
import { usePreferences } from './Preferences.js';
import { WorkspaceRow } from './WorkspaceRow.js';

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
    const [completedFirst, setCompletedFirst] = useState(false);
    const [limit, setLimit] = useState(8);
    const items = sortJobs(jobs, completedFirst);
    return (
        <section
            aria-labelledby='jobs-title'
            className='workspace-section'
            id='view-jobs'
        >
            <div className='workspace-section-heading'>
                <div className='section-title'>
                    <h2 id='jobs-title'>{t('離線任務')}</h2>
                    <button
                        aria-label={t('已完成優先')}
                        aria-pressed={completedFirst}
                        className='icon-button task-sort'
                        id='job-sort'
                        onClick={() => {
                            setCompletedFirst((value) => !value);
                            setLimit(8);
                        }}
                        title={`${t('排序')}：${t(completedFirst ? '已完成優先' : '未完成優先')}`}
                        type='button'
                    >
                        <Icon name='sort' />
                    </button>
                </div>
                <div className='section-actions'>
                    {completedFirst &&
                        jobs.some((job) => job.state === 'complete') && (
                            <button
                                aria-label={t('清除已完成任務')}
                                className='icon-button section-clear'
                                disabled={busy}
                                id='clear-completed-jobs'
                                onClick={onClear}
                                title={t('清除已完成任務')}
                                type='button'
                            >
                                <Icon name='trash' />
                            </button>
                        )}
                    <AddButton kind='task' onClick={onAdd} />
                </div>
            </div>
            <div id='jobs-list'>
                {items.slice(0, limit).map((job) => (
                    <WorkspaceRow
                        id={job.id}
                        key={job.id}
                        kind='job'
                        label={`${job.rule.title} · ${job.title} · ${t(jobLabels[job.state] || job.state)}`}
                        leading={
                            <span
                                className='job-indicator'
                                title={t(jobLabels[job.state] || job.state)}
                            >
                                <span
                                    aria-hidden='true'
                                    className={`dot ${jobColor(job.state) === 'orange' ? 'amber' : jobColor(job.state)}`}
                                />
                            </span>
                        }
                        onOpen={() => {
                            onOpen(job.id);
                        }}
                        title={job.rule.title || job.title}
                    >
                        <div className='job-meta'>
                            <span className='job-filename' title={job.title}>
                                {job.title}
                            </span>
                            <span aria-hidden='true' className='meta-separator'>
                                ·
                            </span>
                            <time>{date(job.created_at)}</time>
                        </div>
                    </WorkspaceRow>
                ))}
                {items.length === 0 && (
                    <p className='workspace-idle'>{t('尚無任務')}</p>
                )}
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
