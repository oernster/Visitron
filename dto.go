package main

// The wire: every shape the page receives or sends. frontend/src/api.ts states
// the same shapes in TypeScript; a structural test compares the two.

// StateDTO is what the page needs to start.
type StateDTO struct {
	Name    string `json:"name"`
	Version string `json:"version"`
	Problem string `json:"problem"`
	Periods []int  `json:"periods"`
}

// WebsiteRowDTO is one row of the website list (FR-041).
type WebsiteRowDTO struct {
	ID             int64    `json:"id"`
	URL            string   `json:"url"`
	Repos          []string `json:"repos"`
	PageLoads      int      `json:"pageLoads"`
	Downloads      int      `json:"downloads"`
	TotalDownloads int      `json:"totalDownloads"`
	SinceLastCheck int      `json:"sinceLastCheck"`
}

// OverviewDTO is the whole window's figures plus the check's state (FR-044).
type OverviewDTO struct {
	Rows        []WebsiteRowDTO `json:"rows"`
	Period      int             `json:"period"`
	LastSuccess string          `json:"lastSuccess"`
	LastFailure string          `json:"lastFailure"`
	Failure     string          `json:"failure"`
	NoKey       bool            `json:"noKey"`
	Running     bool            `json:"running"`
}

// NamedCountDTO is one labelled total.
type NamedCountDTO struct {
	Name  string `json:"name"`
	Count int    `json:"count"`
}

// DayCountDTO is one day's figure.
type DayCountDTO struct {
	Day   string `json:"day"`
	Count int    `json:"count"`
}

// DetailDTO is the selected website's figures (FR-042).
type DetailDTO struct {
	ID             int64           `json:"id"`
	URL            string          `json:"url"`
	Total          int             `json:"total"`
	ByRepo         []NamedCountDTO `json:"byRepo"`
	ByRelease      []NamedCountDTO `json:"byRelease"`
	ByPlatform     []NamedCountDTO `json:"byPlatform"`
	DailyPageLoads []DayCountDTO   `json:"dailyPageLoads"`
	DailyDownloads []DayCountDTO   `json:"dailyDownloads"`
}

// ProposalDTO is what the Add and Edit dialogs offer after a crawl.
type ProposalDTO struct {
	URL     string   `json:"url"`
	Found   []string `json:"found"`
	Ticked  []string `json:"ticked"`
	Problem string   `json:"problem"`
}

// ProgressDTO is sent to the page as a check moves on (FR-033).
type ProgressDTO struct {
	Done  int    `json:"done"`
	Total int    `json:"total"`
	Site  string `json:"site"`
}

// SettingsDTO is the Settings dialog (FR-060).
type SettingsDTO struct {
	IntervalHours    int  `json:"intervalHours"`
	MinInterval      int  `json:"minInterval"`
	MaxInterval      int  `json:"maxInterval"`
	UpdateCheck      bool `json:"updateCheck"`
	StartWithWindows bool `json:"startWithWindows"`
	GoatCounterSet   bool `json:"goatCounterSet"`
	GitHubTokenSet   bool `json:"gitHubTokenSet"`
}

// CreditDTO is one open source credit in About (FR-072).
type CreditDTO struct {
	Work    string `json:"work"`
	Licence string `json:"licence"`
	Holder  string `json:"holder"`
}

// AboutDTO is the About dialog.
type AboutDTO struct {
	Name    string      `json:"name"`
	Author  string      `json:"author"`
	Version string      `json:"version"`
	Licence string      `json:"licence"`
	Credits []CreditDTO `json:"credits"`
}
