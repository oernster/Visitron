package main

import (
	"errors"
	"fmt"
	"os"
	"sort"
	"time"

	"github.com/oernster/visitron/internal/application"
	"github.com/oernster/visitron/internal/domain"
	"github.com/oernster/visitron/internal/licence"
	"github.com/oernster/visitron/internal/product"
)

// The bound methods, one per user-visible action. Each converts the page's
// shape, calls one application service and converts the answer back.

// stampLayout is how the window states a check's time (FR-044).
const stampLayout = "Mon 2 Jan 2006, 15:04"

// State answers what the page needs to start.
func (a *App) State() (state StateDTO, err error) {
	defer guard(&err)
	periods := make([]int, len(domain.Periods))
	for i, p := range domain.Periods {
		periods[i] = int(p)
	}
	return StateDTO{Name: product.Name, Version: a.version, Problem: a.problem, Periods: periods}, nil
}

// Overview answers the website list over the saved period (FR-041, FR-044).
func (a *App) Overview() (overview OverviewDTO, err error) {
	defer guard(&err)
	prefs, err := application.Preferred(a.services.Store)
	if err != nil {
		return OverviewDTO{}, err
	}
	rows, err := a.services.Figures.Overview(prefs.Period)
	if err != nil {
		return OverviewDTO{}, err
	}
	rec, err := a.services.Store.CheckRecord()
	if err != nil {
		return OverviewDTO{}, err
	}
	key, err := a.services.Settings.View()
	if err != nil {
		return OverviewDTO{}, err
	}
	out := OverviewDTO{
		Rows: make([]WebsiteRowDTO, 0, len(rows)), Period: int(prefs.Period),
		LastSuccess: stamp(rec.LastSuccess), Failure: rec.Failure,
		NoKey: !key.GoatCounterSet, Running: a.services.Scheduler.Running(),
	}
	if rec.LastFailure.After(rec.LastSuccess) {
		out.LastFailure = stamp(rec.LastFailure)
	} else {
		out.Failure = ""
	}
	for _, r := range rows {
		out.Rows = append(out.Rows, WebsiteRowDTO{
			ID: r.Website.ID, URL: r.Website.Address.URL(), Repos: repoNames(r.Website.Repos),
			PageLoads: r.PageLoads, Downloads: r.Downloads,
			TotalDownloads: r.TotalDownloads, SinceLastCheck: r.SinceLastCheck,
		})
	}
	return out, nil
}

// Detail answers one website's figures over the saved period (FR-042).
func (a *App) Detail(id int64) (detail DetailDTO, err error) {
	defer guard(&err)
	prefs, err := application.Preferred(a.services.Store)
	if err != nil {
		return DetailDTO{}, err
	}
	d, err := a.services.Figures.Detail(id, prefs.Period)
	if err != nil {
		return DetailDTO{}, err
	}
	platforms := map[string]int{}
	for p, n := range d.Totals.ByPlatform {
		platforms[string(p)] = n
	}
	byPlatform := []NamedCountDTO{}
	for _, p := range domain.Platforms {
		if n, ok := platforms[string(p)]; ok {
			byPlatform = append(byPlatform, NamedCountDTO{Name: string(p), Count: n})
		}
	}
	return DetailDTO{
		ID: id, URL: d.Website.Address.URL(), Total: d.Totals.All,
		ByRepo: named(d.Totals.ByRepo), ByRelease: named(d.Totals.ByRelease), ByPlatform: byPlatform,
		DailyPageLoads: days(d.DailyPageLoads), DailyDownloads: days(d.DailyDownloads),
	}, nil
}

// Propose crawls an address for the Add (editID 0) or Edit dialog (FR-001,
// FR-004).
func (a *App) Propose(entry string, editID int64) (proposal ProposalDTO, err error) {
	defer guard(&err)
	p, err := a.services.Websites.Propose(a.ctx, entry, editID)
	if err != nil {
		return ProposalDTO{}, err
	}
	return ProposalDTO{URL: p.Address.URL(), Found: repoNames(p.Found), Ticked: repoNames(p.Ticked), Problem: p.Problem}, nil
}

// ConfirmRepo checks a repository typed by hand (FR-009).
func (a *App) ConfirmRepo(text string) (name string, err error) {
	defer guard(&err)
	repo, err := a.services.Websites.Confirm(a.ctx, text)
	return repo.String(), err
}

// SaveWebsite records the dialog's website; id 0 adds one.
func (a *App) SaveWebsite(id int64, url string, repos []string) (saved int64, err error) {
	defer guard(&err)
	addr, err := domain.Normalise(url)
	if err != nil {
		return 0, err
	}
	chosen := make([]domain.Repo, 0, len(repos))
	for _, text := range repos {
		repo, err := domain.ParseRepo(text)
		if err != nil {
			return 0, err
		}
		chosen = append(chosen, repo)
	}
	return a.services.Websites.Save(id, addr, chosen)
}

// DeleteWebsite removes a website once the page has confirmed (FR-010).
func (a *App) DeleteWebsite(id int64) (err error) {
	defer guard(&err)
	return a.services.Websites.Delete(id)
}

// Refresh starts a check now on a goroutine of its own and answers at once;
// progress arrives as events (FR-032, FR-033).
func (a *App) Refresh() (err error) {
	defer guard(&err)
	if a.services.Scheduler.Running() {
		return application.ErrBusy
	}
	go func() {
		defer func() {
			if recovered := recover(); recovered != nil {
				fmt.Fprintf(os.Stderr, "panic in a refresh: %v\n", recovered)
			}
			a.emit(progressEvent, ProgressDTO{})
		}()
		if _, err := a.services.Scheduler.Refresh(a.ctx, a.progress); err != nil && !errors.Is(err, application.ErrBusy) {
			fmt.Fprintf(os.Stderr, "refresh: %v\n", err)
		}
	}()
	return nil
}

// Settings answers the Settings dialog (FR-060).
func (a *App) Settings() (settings SettingsDTO, err error) {
	defer guard(&err)
	v, err := a.services.Settings.View()
	if err != nil {
		return SettingsDTO{}, err
	}
	return SettingsDTO{
		IntervalHours: v.IntervalHours, MinInterval: domain.MinIntervalHours, MaxInterval: domain.MaxIntervalHours,
		UpdateCheck: v.UpdateCheck, StartWithWindows: v.StartWithWindows,
		GoatCounterSet: v.GoatCounterSet, GitHubTokenSet: v.GitHubTokenSet,
	}, nil
}

// SaveInterval keeps the check interval (FR-063).
func (a *App) SaveInterval(hours int) (err error) {
	defer guard(&err)
	return a.services.Settings.SaveInterval(hours)
}

// SavePeriod keeps the period the window reports over (FR-043).
func (a *App) SavePeriod(days int) (err error) {
	defer guard(&err)
	return a.services.Settings.SavePeriod(days)
}

// SaveUpdateCheck turns the update check on or off (FR-075).
func (a *App) SaveUpdateCheck(on bool) (err error) {
	defer guard(&err)
	return a.services.Settings.SaveUpdateCheck(on)
}

// SaveStartWithWindows turns the sign-in start on or off (FR-052).
func (a *App) SaveStartWithWindows(on bool) (err error) {
	defer guard(&err)
	return a.services.Settings.SaveStartWithWindows(on)
}

// SaveSecret stores the GoatCounter key ("goatcounter") or GitHub token
// ("github"), answering why it did not work, "" when it did (FR-061).
func (a *App) SaveSecret(which, value string) (problem string, err error) {
	defer guard(&err)
	name, err := secretNamed(which)
	if err != nil {
		return "", err
	}
	return a.services.Settings.SaveSecret(a.ctx, name, value)
}

// RemoveSecret forgets the key or token (FR-062).
func (a *App) RemoveSecret(which string) (err error) {
	defer guard(&err)
	name, err := secretNamed(which)
	if err != nil {
		return err
	}
	return a.services.Settings.RemoveSecret(name)
}

// About answers the About dialog (FR-072).
func (a *App) About() (about AboutDTO, err error) {
	defer guard(&err)
	credits := make([]CreditDTO, len(application.Credits))
	for i, c := range application.Credits {
		credits[i] = CreditDTO(c)
	}
	return AboutDTO{Name: product.Name, Author: product.Author, Version: a.version,
		Licence: licence.Text(), Credits: credits}, nil
}

// Donate opens the donate page in the browser (FR-073).
func (a *App) Donate() (err error) {
	defer guard(&err)
	if a.opener == nil {
		return errors.New("no browser opener is wired")
	}
	a.opener.Open(product.DonateURL)
	return nil
}

var errSecretName = errors.New("there is no such secret")

func secretNamed(which string) (application.Secret, error) {
	switch which {
	case "goatcounter":
		return application.GoatCounterKey, nil
	case "github":
		return application.GitHubToken, nil
	}
	return "", errSecretName
}

func stamp(t time.Time) string {
	if t.IsZero() {
		return ""
	}
	return t.Local().Format(stampLayout)
}

func repoNames(repos []domain.Repo) []string {
	out := make([]string, len(repos))
	for i, r := range repos {
		out[i] = r.String()
	}
	return out
}

// named sorts totals largest first, then by name, for a stable table.
func named(counts map[string]int) []NamedCountDTO {
	out := make([]NamedCountDTO, 0, len(counts))
	for name, n := range counts {
		out = append(out, NamedCountDTO{Name: name, Count: n})
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].Count != out[j].Count {
			return out[i].Count > out[j].Count
		}
		return out[i].Name < out[j].Name
	})
	return out
}

func days(counts []domain.DayCount) []DayCountDTO {
	out := make([]DayCountDTO, len(counts))
	for i, c := range counts {
		out[i] = DayCountDTO{Day: c.Day.String(), Count: c.Count}
	}
	return out
}
