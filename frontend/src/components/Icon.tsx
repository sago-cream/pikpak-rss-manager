import type { JSX } from 'react';
import {
    ArrowRight,
    BrushCleaning,
    Check,
    ChevronDown,
    ChevronRight,
    CircleAlert,
    CircleCheck,
    Clock,
    Download,
    EllipsisVertical,
    Folder,
    Info,
    LoaderCircle,
    Moon,
    Pause,
    Pencil,
    Play,
    Plus,
    RefreshCw,
    Rss,
    Sun,
    Trash2,
    UserRound,
    X,
} from 'lucide-react';
import type { LucideIcon } from 'lucide-react';

export type IconName =
    | 'cleanup'
    | 'tick'
    | 'user'
    | 'alert'
    | 'loading'
    | 'more-vertical'
    | 'play'
    | 'pause'
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
    | 'moon'
    | 'close'
    | 'plus'
    | 'arrow-right'
    | 'chevron-right';

const icons: Record<IconName, LucideIcon> = {
    'cleanup': BrushCleaning,
    'tick': Check,
    'user': UserRound,
    'alert': CircleAlert,
    'loading': LoaderCircle,
    'more-vertical': EllipsisVertical,
    'play': Play,
    'pause': Pause,
    'info': Info,
    'rss': Rss,
    'folder': Folder,
    'clock': Clock,
    'edit': Pencil,
    'check': CircleCheck,
    'backfill': Download,
    'trash': Trash2,
    'refresh': RefreshCw,
    'chevron': ChevronDown,
    'sun': Sun,
    'moon': Moon,
    'close': X,
    'plus': Plus,
    'arrow-right': ArrowRight,
    'chevron-right': ChevronRight,
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
            className={`ui-icon${className ? ` ${className}` : ''}`}
            focusable='false'
            size={24}
            strokeWidth={2}
        />
    );
}
