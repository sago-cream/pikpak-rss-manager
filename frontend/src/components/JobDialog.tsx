import { useEffect, useState } from 'react';
import type { JSX } from 'react';

import { api } from '../lib/api.js';
import { errorMessage, isCancelled } from '../lib/i18n.js';
import { currentFilename, reviewStates } from '../lib/jobs.js';
import { useLifetime } from '../lib/lifetime.js';
import type { JobDetails } from '../lib/types.js';
import { Badge } from './Badge.js';
import { Dialog } from './Dialog.js';
import { Feedback } from './FormField.js';
import { usePreferences } from './Preferences.js';

export function JobDialog({
    jobID,
    onClose,
    onChanged,
}: {
    jobID: string;
    onClose: () => void;
    onChanged: () => Promise<void>;
}): JSX.Element {
    const { t } = usePreferences();
    const active = useLifetime();
    const [details, setDetails] = useState<JobDetails | null>(null);
    const [error, setError] = useState('');
    const [pending, setPending] = useState(false);
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
        setPending(true);
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
            setPending(false);
        }
    };
    return (
        <Dialog id='job-dialog' onClose={onClose} title={t('逐檔整理紀錄')}>
            <div id='job-details'>
                {details && (
                    <>
                        <h3>{details.job.rule.title}</h3>
                        <p className='muted'>{details.job.title}</p>
                        <div className='detail-meta'>
                            <Badge state={details.job.state} />
                            <span>
                                {t('目標：')}
                                {details.job.destination || t('根目錄')}
                            </span>
                        </div>
                        {details.job.error && (
                            <p className='notice'>{t(details.job.error)}</p>
                        )}
                        <dl className='id-list'>
                            <dt>{t('任務識別')}</dt>
                            <dd>{details.job.id}</dd>
                            <dt>{t('PikPak 任務 ID')}</dt>
                            <dd>{details.job.task_id || t('尚未取得')}</dd>
                        </dl>
                        {details.files.length > 0 ? (
                            details.files.map((file) => {
                                const filename = currentFilename(file);
                                return (
                                    <article
                                        className='file-detail'
                                        key={file.file_id}
                                    >
                                        <span
                                            className={`badge ${file.state === 'done' ? 'green' : 'orange'}`}
                                        >
                                            {t(
                                                (
                                                    {
                                                        pending: '等待整理',
                                                        renaming: '改名中',
                                                        done: '已完成',
                                                        review: '保留原名',
                                                    } as Record<string, string>
                                                )[file.state] || file.state
                                            )}
                                        </span>
                                        {file.original_name &&
                                            file.original_name !== filename && (
                                                <p>
                                                    {t('原名：')}
                                                    {file.original_name}
                                                </p>
                                            )}
                                        <strong>
                                            {filename || t('保留原名')}
                                        </strong>
                                        {file.error && (
                                            <small className='error'>
                                                {t(file.error)}
                                            </small>
                                        )}
                                    </article>
                                );
                            })
                        ) : (
                            <p className='muted'>{t('尚無逐檔整理紀錄。')}</p>
                        )}
                        <div className='dialog-footer'>
                            <button
                                className='button subtle danger'
                                data-delete-job={details.job.id}
                                disabled={pending}
                                onClick={() => {
                                    void mutate(true);
                                }}
                                type='button'
                            >
                                {t('刪除任務')}
                            </button>
                            {reviewStates.includes(details.job.state) &&
                                details.job.state !== 'failed' && (
                                    <button
                                        className='button primary'
                                        data-retry={details.job.id}
                                        disabled={pending}
                                        onClick={() => {
                                            void mutate(false);
                                        }}
                                        type='button'
                                    >
                                        {t(
                                            details.job.state ===
                                                'submission_unknown'
                                                ? '重新核對結果'
                                                : '重新整理／接續任務'
                                        )}
                                    </button>
                                )}
                        </div>
                    </>
                )}
                <Feedback message={t(error)} />
            </div>
        </Dialog>
    );
}
