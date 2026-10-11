import type { Job } from './types.js';

export const runningStates = [
    'queued',
    'submitting',
    'downloading',
    'organizing',
];
export const reviewStates = [
    'needs_review',
    'submission_unknown',
    'failed',
    'paused_auth',
    'paused_quota',
    'paused_account',
];
export type JobFilter = 'all' | 'running' | 'review' | 'complete';
export const jobLabels: Record<string, string> = {
    queued: '等待處理',
    submitting: '正在提交',
    submission_unknown: '提交待核對',
    downloading: '雲端下載中',
    organizing: '正在整理',
    complete: '已完成',
    needs_review: '待處理',
    failed: '下載失敗',
    paused_auth: '授權暫停',
    paused_quota: '配額暫停',
    paused_account: '帳號暫停',
};
export function filterJobs(jobs: Job[], filter: JobFilter): Job[] {
    const items = jobs.filter(
        (job) =>
            filter === 'all' ||
            (filter === 'running' && runningStates.includes(job.state)) ||
            (filter === 'review' && reviewStates.includes(job.state)) ||
            (filter === 'complete' && job.state === 'complete')
    );
    const priority = (job: Job) =>
        reviewStates.includes(job.state)
            ? 0
            : runningStates.includes(job.state)
              ? 1
              : 2;
    return filter === 'all'
        ? items.toSorted((a, b) => priority(a) - priority(b))
        : items;
}
export function currentFilename(file: {
    actual_name?: string;
    original_name: string;
    target_name: string;
}): string {
    return file.actual_name || file.original_name || file.target_name;
}
export function formatBytes(value?: string): string {
    let n = Number(value || 0);
    if (!Number.isFinite(n)) {
        return '—';
    }
    const units = ['B', 'KiB', 'MiB', 'GiB', 'TiB'];
    let i = 0;
    while (n >= 1024 && i < 4) {
        n /= 1024;
        i++;
    }
    return `${n.toFixed(i ? 1 : 0)} ${units[i]}`;
}
