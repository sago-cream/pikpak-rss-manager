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
import { ProgressMeter } from './ProgressMeter.js';
import { RowMetadata } from './RowMetadata.js';
import { WorkspacePanel } from './WorkspacePanel.js';
import { WorkspaceRow } from './WorkspaceRow.js';

function DownloadProgress({ job }: { job: Job }): JSX.Element | null {
    const { t } = usePreferences();
    if (job.state !== 'downloading' || !Number.isFinite(job.progress)) {
        return null;
    }
    const percent = Math.min(100, Math.max(0, job.progress));
    return (
        <div className='job-progress'>
            <span aria-hidden='true'>{Math.round(percent)}%</span>
            <ProgressMeter
                label={`${t('下載進度')} ${job.title}`}
                shape='circle'
                value={percent}
            />
        </div>
    );
}

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
    const items = sortJobs(jobs);
    return (
        <WorkspacePanel
            actions={<AddButton kind='task' onClick={onAdd} />}
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
                                trailing={<DownloadProgress job={job} />}
                            >
                                <RowMetadata
                                    items={[
                                        <span title={job.title}>
                                            {job.title}
                                        </span>,
                                        <time>{date(job.created_at)}</time>,
                                    ]}
                                />
                            </WorkspaceRow>
                        ))}
                    </section>
                );
            })}
            {items.length === 0 && (
                <p className='workspace-idle'>{t('尚無任務')}</p>
            )}
        </WorkspacePanel>
    );
}
