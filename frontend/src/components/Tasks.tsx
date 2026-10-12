import { useState } from 'react';
import type { JSX } from 'react';

import {
    jobColor,
    jobGroups,
    jobIcon,
    jobLabels,
    sortJobs,
} from '../lib/jobs.js';
import type { Job } from '../lib/types.js';
import { AddButton } from './AddButton.js';
import { Icon } from './Icon.js';
import { usePreferences } from './Preferences.js';
import { Select } from './Select.js';
import { WorkspacePanel } from './WorkspacePanel.js';
import { WorkspaceRow } from './WorkspaceRow.js';

type JobFilter = 'all' | ReturnType<typeof jobColor>;

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
    const items = sortJobs(jobs).filter(
        (job) => filter === 'all' || jobColor(job.state) === filter
    );
    return (
        <WorkspacePanel
            actions={<AddButton kind='task' onClick={onAdd} />}
            controls={
                <Select
                    aria-label={t('任務狀態')}
                    className='task-state-filter'
                    id='job-state-filter'
                    onValueChange={(value) => {
                        setFilter(value as JobFilter);
                        const list = document.querySelector('#jobs-list');
                        if (list) {
                            list.scrollTop = 0;
                        }
                    }}
                    options={[
                        { value: 'all', label: t('全部') },
                        ...jobGroups.map((group) => ({
                            value: group.color,
                            label: t(group.label),
                        })),
                    ]}
                    value={filter}
                />
            }
            id='view-jobs'
            listId='jobs-list'
            listLabel={t('任務清單')}
            title={t('下載任務')}
            titleId='jobs-title'
        >
            {jobGroups.map((group) => {
                const grouped = items.filter(
                    (job) => jobColor(job.state) === group.color
                );
                if (grouped.length === 0) {
                    return null;
                }
                return (
                    <section
                        aria-labelledby={`jobs-group-${group.color}`}
                        className='job-group'
                        key={group.color}
                    >
                        <div className='job-group-heading'>
                            <h3
                                className='job-group-title'
                                id={`jobs-group-${group.color}`}
                            >
                                {t(group.label)}
                            </h3>
                            {group.color === 'green' && (
                                <button
                                    aria-busy={busy}
                                    aria-label={t('清除已完成任務')}
                                    className='icon-button section-clear'
                                    disabled={busy}
                                    id='clear-completed-jobs'
                                    onClick={onClear}
                                    title={t('清除已完成任務')}
                                    type='button'
                                >
                                    <Icon
                                        className={
                                            busy ? 'is-loading' : undefined
                                        }
                                        name={busy ? 'loading' : 'cleanup'}
                                    />
                                </button>
                            )}
                        </div>
                        {grouped.map((job) => (
                            <WorkspaceRow
                                id={job.id}
                                key={job.id}
                                kind='job'
                                label={`${job.rule.title} · ${job.title} · ${t(jobLabels[job.state] || job.state)}`}
                                leading={
                                    <span
                                        className={`job-indicator ${jobColor(job.state)}`}
                                        title={t(
                                            jobLabels[job.state] || job.state
                                        )}
                                    >
                                        <Icon name={jobIcon(job.state)} />
                                    </span>
                                }
                                onOpen={() => {
                                    onOpen(job.id);
                                }}
                                title={job.rule.title || job.title}
                            >
                                <div className='job-meta'>
                                    <span
                                        className='job-filename'
                                        title={job.title}
                                    >
                                        {job.title}
                                    </span>
                                    <span
                                        aria-hidden='true'
                                        className='meta-separator'
                                    >
                                        ·
                                    </span>
                                    <time>{date(job.created_at)}</time>
                                </div>
                            </WorkspaceRow>
                        ))}
                    </section>
                );
            })}
            {items.length === 0 && (
                <p className='workspace-idle'>
                    {t(jobs.length === 0 ? '尚無任務' : '沒有符合的任務')}
                </p>
            )}
        </WorkspacePanel>
    );
}
