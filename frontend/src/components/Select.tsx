import type { ComponentProps, JSX } from 'react';

import { Icon } from './Icon.js';

export function Select({
    className = '',
    ...props
}: ComponentProps<'select'>): JSX.Element {
    return (
        <span className={`select-control ${className}`}>
            <select {...props} className='select-input' />
            <Icon className='select-chevron' name='chevron' />
        </span>
    );
}
