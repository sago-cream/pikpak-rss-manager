import { useLayoutEffect, useRef } from 'react';
import type { JSX, ReactNode } from 'react';

export function ScrollRegion({
    id,
    label,
    children,
}: {
    id: string;
    label: string;
    children: ReactNode;
}): JSX.Element {
    const region = useRef<HTMLDivElement>(null);
    useLayoutEffect(() => {
        const element = region.current;
        const panel = element?.closest<HTMLElement>('.workspace-section');
        if (!element || !panel) {
            return;
        }
        const syncGutter = () => {
            const width = `${element.offsetWidth - element.clientWidth}px`;
            if (
                panel.style.getPropertyValue('--workspace-scrollbar-width') !==
                width
            ) {
                panel.style.setProperty('--workspace-scrollbar-width', width);
            }
        };
        syncGutter();
        if (typeof ResizeObserver === 'undefined') {
            return;
        }
        const observer = new ResizeObserver(syncGutter);
        observer.observe(element);
        return () => {
            observer.disconnect();
        };
    }, []);
    return (
        <div
            aria-label={label}
            className='workspace-scroll'
            id={id}
            ref={region}
            role='group'
            tabIndex={0}
        >
            {children}
        </div>
    );
}
