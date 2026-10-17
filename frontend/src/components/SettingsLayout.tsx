import type { JSX, ReactNode } from 'react';

import { usePreferences } from './Preferences.js';

export function SettingsLayout({
    id,
    disabled = false,
    sections,
    value,
    onChange,
    children,
}: {
    id: string;
    disabled?: boolean;
    sections: ReadonlyArray<{ id: string; label: string }>;
    value: string;
    onChange: (value: string) => void;
    children: ReactNode;
}): JSX.Element {
    const { t } = usePreferences();
    return (
        <div className='settings-layout'>
            <nav aria-label={t('設定分類')} className='settings-nav'>
                {sections.map((section) => (
                    <button
                        aria-controls={`${id}-${section.id}`}
                        aria-current={value === section.id}
                        className='settings-nav-button'
                        data-settings-section={section.id}
                        disabled={disabled}
                        id={`${id}-nav-${section.id}`}
                        key={section.id}
                        onClick={() => {
                            onChange(section.id);
                        }}
                        type='button'
                    >
                        {t(section.label)}
                    </button>
                ))}
            </nav>
            <div className='settings-content'>{children}</div>
        </div>
    );
}

export function SettingsPanel({
    id,
    section,
    value,
    label,
    children,
}: {
    id: string;
    section: string;
    value: string;
    label?: string;
    children: ReactNode;
}): JSX.Element {
    return (
        <article
            aria-label={label}
            aria-labelledby={label ? undefined : `${id}-nav-${section}`}
            className={`settings-panel${value === section ? '' : ' hidden'}`}
            data-settings-panel={section}
            hidden={value !== section}
            id={`${id}-${section}`}
        >
            {children}
        </article>
    );
}
