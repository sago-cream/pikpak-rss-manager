package model

type Rule struct {
	Title    string `json:"title"`
	Season   int    `json:"season"`
	Regex    string `json:"regex"`
	Template string `json:"template"`
}

type Subscription struct {
	ID              int64  `json:"id"`
	Name            string `json:"name"`
	RSSURL          string `json:"rss_url"`
	Destination     string `json:"destination"`
	Enabled         bool   `json:"enabled"`
	IntervalMinutes int    `json:"interval_minutes"`
	Season          int    `json:"season"`
	Regex           string `json:"regex"`
	Template        string `json:"template"`
	Initialized     bool   `json:"initialized"`
	LastChecked     int64  `json:"last_checked"`
	NextCheck       int64  `json:"next_check"`
	LastError       string `json:"last_error"`
}

func (s Subscription) Rule() Rule { return Rule{s.Name, s.Season, s.Regex, s.Template} }

type Job struct {
	ID             string  `json:"id"`
	SubscriptionID int64   `json:"subscription_id"`
	AccountID      string  `json:"-"`
	ResourceKey    string  `json:"resource_key"`
	ResourceURL    string  `json:"-"`
	Title          string  `json:"title"`
	Rule           Rule    `json:"rule"`
	Destination    string  `json:"destination"`
	DestinationID  string  `json:"destination_id"`
	StagingID      string  `json:"staging_id"`
	TaskID         string  `json:"task_id"`
	FileID         string  `json:"file_id"`
	State          string  `json:"state"`
	Progress       float64 `json:"progress"`
	PrimaryCount   int     `json:"primary_count"`
	FileCount      int     `json:"file_count"`
	Attempts       int     `json:"attempts"`
	NextAttempt    int64   `json:"next_attempt"`
	CreatedAt      int64   `json:"created_at"`
	UpdatedAt      int64   `json:"updated_at"`
	Error          string  `json:"error"`
}

// JobRecord is internal persistence; private fields are never exposed by the API.
type JobRecord struct {
	Job
	StoredAccountID   string `json:"account_id"`
	StoredResourceURL string `json:"resource_url"`
}

type FileAction struct {
	JobID         string `json:"job_id"`
	FileID        string `json:"file_id"`
	OriginalName  string `json:"original_name"`
	RelativePath  string `json:"relative_path"`
	TargetName    string `json:"target_name"`
	DestinationID string `json:"destination_id"`
	State         string `json:"state"`
	Error         string `json:"error"`
}

type Event struct {
	ID             int64  `json:"id"`
	JobID          string `json:"job_id"`
	SubscriptionID int64  `json:"subscription_id"`
	Level          string `json:"level"`
	Message        string `json:"message"`
	CreatedAt      int64  `json:"created_at"`
}
