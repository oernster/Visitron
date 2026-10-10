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
	NoToken     bool            `json:"noToken"`
	Running     bool            `json:"running"`
	// Since is the day the period's downloads are counted from while
	// Visitron's history is shorter than the period, "" once it is not
	// (Amendment 9).
	Since string `json:"since"`
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
	ID             int64         `json:"id"`
	URL            string        `json:"url"`
	Total          int           `json:"total"`
	DailyPageLoads []DayCountDTO `json:"dailyPageLoads"`
	DailyDownloads []DayCountDTO `json:"dailyDownloads"`
}

// CountryDTO is one country's visitors (FR-047).
type CountryDTO struct {
	Code  string `json:"code"`
	Name  string `json:"name"`
	Count int    `json:"count"`
}

// StatisticsDTO is one website's Statistics dialog (FR-045 to FR-049): its
// downloads to date by platform, repository and release, then its visitors
// by country over the period. Countries is empty, not absent, when there are
// none; NoGoatCounter and CountriesProblem say why they were not read.
type StatisticsDTO struct {
	ID               int64           `json:"id"`
	URL              string          `json:"url"`
	Period           int             `json:"period"`
	ByPlatform       []NamedCountDTO `json:"byPlatform"`
	ByRepo           []NamedCountDTO `json:"byRepo"`
	ByRelease        []NamedCountDTO `json:"byRelease"`
	Countries        []CountryDTO    `json:"countries"`
	NoGoatCounter    bool            `json:"noGoatCounter"`
	CountriesProblem string          `json:"countriesProblem"`
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
	// GoatCounterSite is the code of the owner's GoatCounter site, "" until
	// set (Amendment 11).
	GoatCounterSite string `json:"goatCounterSite"`
	// SelfDownloads is the owner's own downloads of each macOS disk image,
	// taken off its count, within MaxSelfDownloads (Amendment 19).
	SelfDownloads    int `json:"selfDownloads"`
	MaxSelfDownloads int `json:"maxSelfDownloads"`
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

// UpdateDTO is what one update check found: available, current, skipped,
// unreachable, uncomparable or off; the running version; the release's, empty
// where it could not be read. No address crosses: Download acts on the release
// the facade offered (Amendment 5).
type UpdateDTO struct {
	Outcome string `json:"outcome"`
	Running string `json:"running"`
	Latest  string `json:"latest"`
}
