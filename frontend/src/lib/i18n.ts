import { messageUpdates, translations } from './translations.js';
import type { Language } from './types.js';

const escapePattern = (value: string) =>
    value.replaceAll(/[$()*+.?[\\\]^{|}]/g, String.raw`\$&`);
const patterns = Object.entries(translations)
    .filter(([key]) => /%[ds]/.test(key))
    .map(([key, value]) => ({
        pattern: new RegExp(
            `^${key.split(/%[ds]/).map(escapePattern).join('(.*?)')}$`
        ),
        value,
    }));
export function translate(text: string, language: Language): string {
    const value = messageUpdates[text] ?? text;
    if (language !== 'en') {
        return value;
    }
    if (Object.hasOwn(translations, value)) {
        return translations[value];
    }
    for (const entry of patterns) {
        const match = value.match(entry.pattern);
        if (match) {
            let index = 0;
            return entry.value.replaceAll(/%[ds]/g, () => {
                index++;
                return match[index];
            });
        }
    }
    return value;
}
export function errorMessage(error: unknown): string {
    return error instanceof Error ? error.message : '操作未完成，請稍後再試';
}
export function isCancelled(error: unknown): boolean {
    return error instanceof Error && error.name === 'AbortError';
}
