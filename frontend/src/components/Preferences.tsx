import {
    createContext,
    useCallback,
    useContext,
    useEffect,
    useState,
} from 'react';
import type { JSX, ReactNode } from 'react';

import { translate } from '../lib/i18n.js';
import type { Language, Theme } from '../lib/types.js';
import { Icon } from './Icon.js';

function saved(key: string): string | null {
    try {
        return localStorage.getItem(key);
    } catch {
        return null;
    }
}

function persist(key: string, value: string) {
    try {
        localStorage.setItem(key, value);
    } catch {
        /* Preferences still work when browser storage is unavailable. */
    }
}
const themeMedia = globalThis.matchMedia('(prefers-color-scheme: dark)');
const root = document.querySelector<HTMLElement>('#root');
const initialLanguage = root?.dataset.language || saved('language');
const initialTheme = root?.dataset.theme || saved('theme');
const PreferencesContext = createContext<{
    language: Language;
    theme: Theme;
    t: (text: string) => string;
    setLanguage: (value: Language) => void;
    toggleTheme: () => void;
} | null>(null);
export function Preferences({
    children,
}: {
    children: ReactNode;
}): JSX.Element {
    const [language, setLanguageState] = useState<Language>(
        initialLanguage === 'en' ? 'en' : 'zh-Hant'
    );
    const [theme, setTheme] = useState<Theme>(
        initialTheme === 'dark' ||
            (initialTheme !== 'light' && themeMedia.matches)
            ? 'dark'
            : 'light'
    );
    const t = useCallback(
        (text: string) => translate(text, language),
        [language]
    );
    const setLanguage = (value: Language) => {
        persist('language', value);
        setLanguageState(value);
    };
    const toggleTheme = () => {
        setTheme((value) => {
            const next = value === 'dark' ? 'light' : 'dark';
            persist('theme', next);
            return next;
        });
    };
    useEffect(() => {
        document.documentElement.lang = language;
    }, [language]);
    useEffect(() => {
        document.documentElement.dataset.theme = theme;
    }, [theme]);
    useEffect(() => {
        const systemChange = () => {
            if (!saved('theme') && !root?.dataset.theme) {
                setTheme(themeMedia.matches ? 'dark' : 'light');
            }
        };
        const storageChange = (event: StorageEvent) => {
            if (event.key === 'language' || event.key === null) {
                setLanguageState(saved('language') === 'en' ? 'en' : 'zh-Hant');
            }
            if (event.key === 'theme' || event.key === null) {
                const value = saved('theme');
                setTheme(
                    value === 'dark' ||
                        (value !== 'light' && themeMedia.matches)
                        ? 'dark'
                        : 'light'
                );
            }
        };
        themeMedia.addEventListener('change', systemChange);
        globalThis.addEventListener('storage', storageChange);
        return () => {
            themeMedia.removeEventListener('change', systemChange);
            globalThis.removeEventListener('storage', storageChange);
        };
    }, []);
    return (
        <PreferencesContext
            value={{ language, theme, t, setLanguage, toggleTheme }}
        >
            {children}
        </PreferencesContext>
    );
}
export function usePreferences(): NonNullable<
    React.ContextType<typeof PreferencesContext>
> {
    const value = useContext(PreferencesContext);
    if (!value) {
        throw new Error('Preferences provider missing');
    }
    return value;
}
export function Appearance({
    toggleLanguage = false,
}: {
    toggleLanguage?: boolean;
}): JSX.Element {
    const { language, theme, t, setLanguage, toggleTheme } = usePreferences();
    const label = t(theme === 'dark' ? '切換淺色模式' : '切換深色模式');
    return (
        <div className='appearance-controls'>
            <button
                aria-label={label}
                aria-pressed={theme === 'dark'}
                className='theme-toggle'
                onClick={toggleTheme}
                title={label}
                type='button'
            >
                <Icon name={theme === 'dark' ? 'sun' : 'moon'} />
            </button>
            {toggleLanguage ? (
                <button
                    aria-label={t(
                        language === 'en' ? '切換為繁體中文' : '切換為英文'
                    )}
                    className='language-toggle'
                    lang={language === 'en' ? 'zh-Hant' : 'en'}
                    onClick={() => {
                        setLanguage(language === 'en' ? 'zh-Hant' : 'en');
                    }}
                    type='button'
                >
                    {language === 'en' ? '繁體中文' : 'English'}
                </button>
            ) : (
                <select
                    aria-label={t('語言')}
                    className='language-select'
                    onChange={(event) => {
                        setLanguage(
                            event.target.value === 'en' ? 'en' : 'zh-Hant'
                        );
                    }}
                    value={language}
                >
                    <option value='zh-Hant'>繁體中文</option>
                    <option value='en'>English</option>
                </select>
            )}
        </div>
    );
}
