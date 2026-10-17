export type Page = 'setup' | 'login' | 'onboarding' | 'app' | 'recovery';
export type Language = 'zh-Hant' | 'en';
export type Theme = 'light' | 'dark';
export interface Session {
    initialized: boolean;
    authenticated: boolean;
    csrf: string;
    version: string;
}
export interface Rule {
    title: string;
    regex: string;
    rename_enabled: boolean;
    replacement: string;
}
export interface Subscription extends Omit<Rule, 'title'> {
    id: number;
    name: string;
    rss_url: string;
    destination: string;
    destination_id?: string;
    destination_account_ref?: string;
    enabled: boolean;
    interval_minutes: number;
    initialized: boolean;
    last_checked: number;
    next_check: number;
    last_error: string;
}
export interface Job {
    id: string;
    subscription_id: number;
    title: string;
    rule: Rule;
    destination: string;
    task_id: string;
    state: string;
    progress: number;
    created_at: number;
    error: string;
    submission_pending?: boolean;
}
export interface FileAction {
    file_id: string;
    state: string;
    original_name: string;
    target_name: string;
    actual_name?: string;
    error: string;
}
export interface JobDetails {
    job: Job;
    files: FileAction[];
}
export interface LogEntry {
    id: number;
    level: string;
    message: string;
    job_id: string;
    subscription_id: number;
    created_at: number;
}
export interface Status {
    connected?: boolean;
    paused?: string;
    source?: string;
    name?: string;
    message?: string;
    storage_used?: string;
    storage_total?: string;
    storage_updated_at?: number;
}
export interface AppSettings {
    public_url: string;
    allow_private_feeds: boolean;
}
export interface WorkspaceData {
    subscriptions: Subscription[];
    jobs: Job[];
    events: LogEntry[];
    status: Status;
}
export interface Folder {
    id: string;
    name: string;
    path: string;
}
export interface FolderListing {
    current: Folder;
    breadcrumbs: Folder[];
    folders: Folder[];
    next_token?: string;
    account_ref: string;
}
export interface SourceSample {
    kind: string;
    title: string;
    filename: string;
}
export interface SampleResult {
    items: SourceSample[];
    notices: string[];
}
export interface RenameResult {
    name: string;
    raw_name?: string;
    warnings?: string[];
}
export interface BackfillItem {
    id: string;
    title: string;
    filenames?: string[];
    error?: string;
}
export interface BackfillPlan {
    token: string;
    items: BackfillItem[];
    notices: string[];
}
