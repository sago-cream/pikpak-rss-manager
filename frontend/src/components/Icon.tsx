import type { JSX } from 'react';
import {
    ChevronDown,
    CircleCheck,
    Clock,
    Download,
    Folder,
    Info,
    Moon,
    Pencil,
    RefreshCw,
    Rss,
    Sun,
    Trash2,
} from 'lucide-react';
import type { LucideIcon } from 'lucide-react';

export type IconName =
    | 'info'
    | 'rss'
    | 'folder'
    | 'clock'
    | 'edit'
    | 'check'
    | 'backfill'
    | 'trash'
    | 'refresh'
    | 'chevron'
    | 'sun'
    | 'moon';

const icons: Record<IconName, LucideIcon> = {
    info: Info,
    rss: Rss,
    folder: Folder,
    clock: Clock,
    edit: Pencil,
    check: CircleCheck,
    backfill: Download,
    trash: Trash2,
    refresh: RefreshCw,
    chevron: ChevronDown,
    sun: Sun,
    moon: Moon,
};

export function Icon({
    name,
    className,
}: {
    name: IconName;
    className?: string;
}): JSX.Element {
    const Glyph = icons[name];
    return (
        <Glyph
            aria-hidden='true'
            className={className}
            focusable='false'
            size={24}
            strokeWidth={2}
        />
    );
}
