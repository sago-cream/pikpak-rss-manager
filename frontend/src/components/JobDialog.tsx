import { useEffect, useState } from 'react';
import type { JSX } from 'react';

import { api } from '../lib/api.js';
import { errorMessage, isCancelled } from '../lib/i18n.js';
import { currentFilename, reviewStates } from '../lib/jobs.js';
import { useLifetime } from '../lib/lifetime.js';
import type { FileAction, JobDetails } from '../lib/types.js';
import { Badge } from './Badge.js';
import { Dialog } from './Dialog.js';
import { Feedback } from './FormField.js';
import { Icon } from './Icon.js';
import { usePreferences } from './Preferences.js';

export function JobDialog({
    jobID,
    onClose,
    onChanged,
    onFixConnection,
    connected = false,
}: {
    jobID: string;
    onClose: () => void;
    onChanged: () => Promise<void>;
    onFixConnection?: () => void;
    connected?: boolean;
}): JSX.Element {
    const { t } = usePreferences();
    const active = useLifetime();
    const [details, setDetails] = useState<JobDetails | null>(null);
    const [error, setError] = useState('');
    const [pending, setPending] = useState('');
    useEffect(() => {
        const controller = new AbortController();
        void api<JobDetails>(
            `/api/jobs/${encodeURIComponent(jobID)}`,
            'GET',
            undefined,
            controller.signal
        )
            .then((value) => {
                if (!controller.signal.aborted) {
                    setDetails(value);
                }
            })
            .catch((error_: unknown) => {
                if (!controller.signal.aborted && !isCancelled(error_)) {
                    setError(errorMessage(error_));
                }
            });
        return () => {
            controller.abort();
        };
    }, [jobID]);
    const job = details?.job;
    const mutate = async (remove: boolean) => {
        if (!job || pending) {
            return;
        }
        const local =
            !details.job.task_id &&
            ((details.job.submission_pending ?? false) ||
                ['submission_unknown', 'submitting'].includes(
                    details.job.state
                ));
        if (
            remove &&
            !globalThis.confirm(
                t(
                    local
                        ? '尚無 PikPak 任務 ID，無法自動取消。請先至 PikPak 取消，再刪除此紀錄？'
                        : '刪除這筆任務並取消 PikPak 下載？已完成的雲端檔案會保留。'
                )
            )
        ) {
            return;
        }
        setPending(remove ? 'delete' : 'retry');
        setError('');
        try {
            await api(
                `/api/jobs/${encodeURIComponent(details.job.id)}${remove ? '' : '/retry'}`,
                remove ? 'DELETE' : 'POST',
                remove ? { local_only: local } : {}
            );
            await onChanged();
            if (active.current) {
                onClose();
            }
        } catch (error_) {
            setError(errorMessage(error_));
        } finally {
            setPending('');
        }
    };
    const paused = job?.state.startsWith('paused_');
    const retry =
        job && reviewStates.includes(job.state) && job.state !== 'failed';
    const retryLabel =
        job?.submission_pending || job?.state === 'submission_unknown'
            ? '重新核對結果'
            : job?.state === 'needs_review'
              ? '重試改名'
              : '接續下載';
    const attention =
        details?.files.filter((file) => file.state !== 'done') ?? [];
    const completed =
        details?.files.filter((file) => file.state === 'done') ?? [];
    return (
        <Dialog
            id='job-dialog'
            onClose={onClose}
            scroll
            title={job?.rule.title || job?.title || t('任務詳細資料')}
        >
            <div id='job-details'>
                {job && (
                    <>
                        <p className='muted task-detail-filename'>
                            {job.title}
                        </p>
                        <div className='detail-meta'>
                            <Badge state={job.state} />
                            {job.state === 'downloading' && (
                                <span className='task-progress'>
                                    {Math.round(job.progress)}%
                                </span>
                            )}
                            <span>
                                {t('目標：')}
                                {job.destination || t('根目錄')}
                            </span>
                        </div>
                        {job.error && <p className='notice'>{t(job.error)}</p>}
                        <Feedback message={t(error)} />
                        <div className='form-actions task-recovery'>
                            {paused && onFixConnection && (
                                <button
                                    className={`button ${connected ? 'subtle' : 'primary'}`}
                                    onClick={onFixConnection}
                                    type='button'
                                >
                                    <Icon name='alert' />
                                    {t('修復連線')}
                                </button>
                            )}
                            {retry && (!paused || connected) && (
                                <button
                                    aria-busy={pending === 'retry'}
                                    className='button primary'
                                    data-retry={job.id}
                                    disabled={Boolean(pending)}
                                    onClick={() => {
                                        void mutate(false);
                                    }}
                                    type='button'
                                >
                                    <Icon
                                        className={
                                            pending === 'retry'
                                                ? 'is-loading'
                                                : undefined
                                        }
                                        name={
                                            pending === 'retry'
                                                ? 'loading'
                                                : 'refresh'
                                        }
                                    />
                                    {t(retryLabel)}
                                </button>
                            )}
                        </div>
                        {attention.map((file) => (
                            <FileDetail file={file} key={file.file_id} />
                        ))}
                        {completed.length > 0 && (
                            <details className='task-completed-files'>
                                <summary>
                                    {t('已完成')} · {completed.length}
                                </summary>
                                {completed.map((file) => (
                                    <FileDetail
                                        file={file}
                                        key={file.file_id}
                                    />
                                ))}
                            </details>
                        )}
                        {details.files.length === 0 && (
                            <p className='muted'>{t('尚無逐檔整理紀錄。')}</p>
                        )}
                        <details className='technical-details'>
                            <summary>{t('技術資訊')}</summary>
                            <dl className='id-list'>
                                <dt>{t('任務識別')}</dt>
                                <dd>{job.id}</dd>
                                <dt>{t('PikPak 任務 ID')}</dt>
                                <dd>{job.task_id || t('尚未取得')}</dd>
                            </dl>
                        </details>
                        <div className='dialog-footer'>
                            <button
                                aria-busy={pending === 'delete'}
                                className='button subtle danger'
                                data-delete-job={job.id}
                                disabled={Boolean(pending)}
                                onClick={() => {
                                    void mutate(true);
                                }}
                                type='button'
                            >
                                {pending === 'delete' && (
                                    <Icon
                                        className='is-loading'
                                        name='loading'
                                    />
                                )}
                                {t('刪除任務')}
                            </button>
                        </div>
                    </>
                )}
                {!details && <Feedback message={t(error)} />}
                {!details && !error && (
                    <p role='status'>{t('正在讀取任務…')}</p>
                )}
            </div>
        </Dialog>
    );
}
function FileDetail({ file }: { file: FileAction }): JSX.Element {
    const { t } = usePreferences();
    const filename = currentFilename(file);
    const labels: Record<string, string> = {
        pending: '等待整理',
        renaming: '改名中',
        done: '已完成',
        review: '保留原名',
    };
    return (
        <article className='file-detail'>
            <span
                className={`badge ${file.state === 'done' ? 'green' : 'warning'}`}
            >
                {t(labels[file.state] || file.state)}
            </span>
            {file.original_name && file.original_name !== filename && (
                <p>
                    {t('原名：')}
                    {file.original_name}
                </p>
            )}
            <strong>{filename || t('保留原名')}</strong>
            {file.error && <small className='error'>{t(file.error)}</small>}
        </article>
    );
}
