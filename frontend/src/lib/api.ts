let csrf = '';
export class APIError extends Error {
    readonly status: number;
    constructor(message: string, status: number) {
        super(message);
        this.name = 'APIError';
        this.status = status;
    }
}
export function setCSRF(value: string): void {
    csrf = value;
}
export async function api<T>(
    path: string,
    method = 'GET',
    body?: unknown,
    signal?: AbortSignal
): Promise<T> {
    const controller = new AbortController();
    const abort = () => {
        controller.abort();
    };
    if (signal?.aborted) {
        abort();
    } else {
        signal?.addEventListener('abort', abort, { once: true });
    }
    const timeout = setTimeout(abort, 110_000);
    try {
        const response = await fetch(path, {
            method,
            credentials: 'same-origin',
            signal: controller.signal,
            headers: {
                'Content-Type': 'application/json',
                'X-CSRF-Token': csrf,
            },
            body: body === undefined ? undefined : JSON.stringify(body),
        });
        const result: unknown = await response.json();
        if (!response.ok) {
            if (
                response.status === 401 &&
                !['/api/login', '/api/password/recovery'].includes(path)
            ) {
                globalThis.dispatchEvent(new Event('session-expired'));
            }
            const message =
                result !== null &&
                typeof result === 'object' &&
                'error' in result &&
                typeof result.error === 'string'
                    ? result.error
                    : '操作未完成，請稍後再試';
            throw new APIError(message, response.status);
        }
        return result as T;
    } catch (error) {
        if (
            error instanceof Error &&
            error.name === 'AbortError' &&
            !signal?.aborted
        ) {
            throw new Error('操作逾時，請檢查任務紀錄後再試', { cause: error });
        }
        throw error;
    } finally {
        clearTimeout(timeout);
        signal?.removeEventListener('abort', abort);
    }
}
