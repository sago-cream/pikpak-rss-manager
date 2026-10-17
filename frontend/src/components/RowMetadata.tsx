import { Fragment } from 'react';
import type { JSX, ReactNode } from 'react';

export function RowMetadata({ items }: { items: ReactNode[] }): JSX.Element {
    return (
        <div className='row-metadata'>
            {items.map((item, index) => (
                <Fragment key={index}>
                    {index > 0 && ' '}
                    <span className='row-metadata-item'>
                        {index > 0 && <span aria-hidden='true'>{'· '}</span>}
                        {item}
                    </span>
                </Fragment>
            ))}
        </div>
    );
}
