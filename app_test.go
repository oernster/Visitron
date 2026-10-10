package main

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"visitron/internal/application"
	"visitron/internal/domain"
	"visitron/internal/infrastructure/store"
	"visitron/internal/product"
)

func TestStateNamesTheProductAndEveryPeriod(t *testing.T) {
	r := newRig(t, nil)
	r.app.problem = "the data could not be opened"
	state, err := r.app.State()
	if err != nil {
		t.Fatal(err)
	}
	if state.Name != product.Name || state.Version != "1.2.3" || state.Problem != r.app.problem {
		t.Fatalf("state = %+v", state)
	}
	if len(state.Periods) != len(domain.Periods) || state.Periods[0] != int(domain.Week) {
		t.Fatalf("periods = %v", state.Periods)
	}
}

func TestOverviewOfNothingSaysNotCheckedAndNoKey(t *testing.T) {
	r := newRig(t, nil)
	o, err := r.app.Overview()
	if err != nil {
		t.Fatal(err)
	}
	if len(o.Rows) != 0 || o.LastSuccess != "" || o.LastFailure != "" || !o.NoKey || o.Running {
		t.Fatalf("overview = %+v", o)
	}
	if o.Period != int(domain.DefaultPeriod) {
		t.Fatalf("period = %d, want the default", o.Period)
	}
}

func TestOverviewStatesOnlyAFailureNewerThanTheLastSuccess(t *testing.T) {
	r := newRig(t, nil)
	success := time.Date(2026, time.October, 9, 20, 0, 0, 0, time.Local)
	rec := application.CheckRecord{LastSuccess: success, LastFailure: success.Add(-time.Hour), Failure: "offline"}
	if err := r.store.SaveCheckRecord(rec); err != nil {
		t.Fatal(err)
	}
	o, _ := r.app.Overview()
	if o.LastSuccess != "Fri 9 Oct 2026, 20:00" || o.LastFailure != "" || o.Failure != "" {
		t.Fatalf("an older failure is shown: %+v", o)
	}

	rec.LastFailure = success.Add(time.Hour)
	if err := r.store.SaveCheckRecord(rec); err != nil {
		t.Fatal(err)
	}
	o, _ = r.app.Overview()
	if o.LastFailure != "Fri 9 Oct 2026, 21:00" || o.Failure != "offline" {
		t.Fatalf("a newer failure is not shown: %+v", o)
	}
}

func TestSaveCheckThenOverviewAndDetail(t *testing.T) {
	r := newRig(t, nil)
	id, err := r.app.SaveWebsite(0, "example.org", []string{widgetRepo.String()})
	if err != nil || id == 0 {
		t.Fatalf("SaveWebsite = %d, %v", id, err)
	}
	r.app.tickOnce()

	o, err := r.app.Overview()
	if err != nil || len(o.Rows) != 1 {
		t.Fatalf("Overview = %+v, %v", o, err)
	}
	row := o.Rows[0]
	if row.ID != id || row.URL != "https://example.org/" || row.Repos[0] != widgetRepo.String() {
		t.Fatalf("row = %+v", row)
	}
	if o.LastSuccess == "" {
		t.Fatal("a check that succeeded is not stated")
	}
	if o.Since != "9 Oct" || !o.NoKey || !o.NoToken {
		t.Fatalf("since %q, no key %v, no token %v; want counting from the one check, both missing",
			o.Since, o.NoKey, o.NoToken)
	}

	d, err := r.app.Detail(id)
	if err != nil {
		t.Fatal(err)
	}
	if d.Total != row.TotalDownloads || d.Total == 0 {
		t.Fatalf("detail total %d, overview total %d", d.Total, row.TotalDownloads)
	}
	if len(d.ByPlatform) != 2 || d.ByPlatform[0].Name != string(domain.Windows) || d.ByPlatform[1].Name != string(domain.MacOS) {
		t.Fatalf("platforms are not in the domain's order: %+v", d.ByPlatform)
	}
	if len(d.ByRepo) != 1 || len(d.ByRelease) != 1 || len(d.DailyDownloads) != int(domain.DefaultPeriod) {
		t.Fatalf("detail = %+v", d)
	}
	if events := r.window.seen(); len(events) == 0 || events[len(events)-1] != (ProgressDTO{}) {
		t.Fatalf("the tick did not end with an empty progress: %v", events)
	}
}

func TestSaveWebsiteRefusesABadAddressOrRepository(t *testing.T) {
	r := newRig(t, nil)
	if _, err := r.app.SaveWebsite(0, " ", nil); !errors.Is(err, domain.ErrEmptyAddress) {
		t.Fatalf("empty address: %v", err)
	}
	if _, err := r.app.SaveWebsite(0, "example.org", []string{"no-slash"}); !errors.Is(err, domain.ErrRepoForm) {
		t.Fatalf("bad repository: %v", err)
	}
}

func TestEditAndDeleteAWebsite(t *testing.T) {
	r := newRig(t, nil)
	id, _ := r.app.SaveWebsite(0, "example.org", nil)
	if saved, err := r.app.SaveWebsite(id, "example.org", []string{widgetRepo.String()}); err != nil || saved != id {
		t.Fatalf("edit = %d, %v", saved, err)
	}
	if err := r.app.DeleteWebsite(id); err != nil {
		t.Fatal(err)
	}
	if o, _ := r.app.Overview(); len(o.Rows) != 0 {
		t.Fatalf("the website is still listed: %+v", o.Rows)
	}
}

func TestProposeAndConfirmPassThroughTheirRefusals(t *testing.T) {
	r := newRig(t, nil)
	p, err := r.app.Propose("example.org", 0)
	if err != nil || p.URL != "https://example.org/" || p.Problem == "" {
		t.Fatalf("a site that cannot be read: %+v, %v", p, err)
	}
	if _, err := r.app.Propose("", 0); err == nil {
		t.Fatal("an empty address was proposed")
	}
	if name, err := r.app.ConfirmRepo(widgetRepo.String()); err != nil || name != widgetRepo.String() {
		t.Fatalf("ConfirmRepo = %q, %v", name, err)
	}
}

func TestSettingsRoundTrip(t *testing.T) {
	r := newRig(t, nil)
	steps := []error{
		r.app.SaveInterval(domain.MinIntervalHours),
		r.app.SavePeriod(int(domain.Week)),
		r.app.SaveUpdateCheck(false),
		r.app.SaveStartWithWindows(true),
		r.app.SaveSelfDownloads(1),
	}
	for i, err := range steps {
		if err != nil {
			t.Fatalf("step %d: %v", i, err)
		}
	}
	if problem, err := r.app.SaveGoatCounterSite("someone.goatcounter.com"); err != nil || problem != "" {
		t.Fatalf("SaveGoatCounterSite = %q, %v", problem, err)
	}
	if problem, err := r.app.SaveSecret("goatcounter", "key"); err != nil || problem != "" {
		t.Fatalf("SaveSecret = %q, %v", problem, err)
	}
	s, err := r.app.Settings()
	if err != nil {
		t.Fatal(err)
	}
	want := SettingsDTO{IntervalHours: domain.MinIntervalHours, MinInterval: domain.MinIntervalHours,
		MaxInterval: domain.MaxIntervalHours, StartWithWindows: true, GoatCounterSet: true,
		GoatCounterSite: "someone", SelfDownloads: 1, MaxSelfDownloads: domain.MaxSelfDownloads}
	if s != want {
		t.Fatalf("settings = %+v, want %+v", s, want)
	}
	if key, err := r.app.Secret("goatcounter"); key != "key" || err != nil {
		t.Fatalf("Secret = %q, %v", key, err)
	}
	if err := r.app.RemoveSecret("goatcounter"); err != nil || r.secrets.values[application.GoatCounterKey] != "" {
		t.Fatalf("RemoveSecret: %v", err)
	}
	if key, err := r.app.Secret("goatcounter"); key != "" || err != nil {
		t.Fatalf("Secret after removal = %q, %v; want none", key, err)
	}
}

func TestAnUnknownSecretIsRefused(t *testing.T) {
	r := newRig(t, nil)
	if _, err := r.app.SaveSecret("password", "x"); !errors.Is(err, errSecretName) {
		t.Fatalf("SaveSecret: %v", err)
	}
	if err := r.app.RemoveSecret("password"); !errors.Is(err, errSecretName) {
		t.Fatalf("RemoveSecret: %v", err)
	}
	if _, err := r.app.Secret("password"); !errors.Is(err, errSecretName) {
		t.Fatalf("Secret: %v", err)
	}
	if err := r.app.RemoveSecret("github"); err != nil {
		t.Fatalf("the GitHub token is not a known secret: %v", err)
	}
}

func TestAboutCarriesTheLicenceAndCredits(t *testing.T) {
	r := newRig(t, nil)
	a, err := r.app.About()
	if err != nil || a.Version != "1.2.3" || len(a.Credits) != len(application.Credits) {
		t.Fatalf("About = %+v, %v", a, err)
	}
	if !strings.Contains(a.Licence, "GNU GENERAL PUBLIC LICENSE") {
		t.Fatal("the licence text is missing")
	}
}

func TestDonateOpensTheOneAddressOrSaysWhyNot(t *testing.T) {
	r := newRig(t, nil)
	if err := r.app.Donate(); err != nil || len(r.window.opened) != 1 || r.window.opened[0] != product.DonateURL {
		t.Fatalf("Donate: %v, opened %v", err, r.window.opened)
	}
	r.app.opener = nil
	if err := r.app.Donate(); err == nil {
		t.Fatal("a missing opener was not stated")
	}
}

func TestEveryReadSaysWhyTheDataIsUnavailable(t *testing.T) {
	r := newRig(t, store.Unavailable{Reason: errPlanted})
	if _, err := r.app.Overview(); !errors.Is(err, errPlanted) {
		t.Fatalf("Overview: %v", err)
	}
	if _, err := r.app.Detail(1); !errors.Is(err, errPlanted) {
		t.Fatalf("Detail: %v", err)
	}
	if _, err := r.app.Settings(); !errors.Is(err, errPlanted) {
		t.Fatalf("Settings: %v", err)
	}
}

func TestAPanicInABoundMethodBecomesASentence(t *testing.T) {
	r := newRig(t, nil)
	r.app.services.Websites = nil
	if err := r.app.DeleteWebsite(1); !errors.Is(err, errInternal) {
		t.Fatalf("err = %v, want errInternal", err)
	}
}

func TestAPanicInTheSchedulerIsSurvived(t *testing.T) {
	r := newRig(t, nil)
	r.app.services.Scheduler = nil
	r.app.tickOnce() // must return rather than end the test binary
	if events := r.window.seen(); len(events) != 1 || events[0] != (ProgressDTO{}) {
		t.Fatalf("the page was not told the tick ended: %v", events)
	}
}

func TestAPanicInACheckIsRecordedAsAFailedCheck(t *testing.T) {
	r := newRig(t, nil)
	// The panic is planted where a check reports its progress.
	if _, err := r.app.SaveWebsite(0, "example.org", nil); err != nil {
		t.Fatal(err)
	}
	r.app.emitter = panicker{}
	r.app.tickOnce()
	o, err := r.app.Overview()
	if err != nil || o.LastFailure == "" || o.Failure != errInternal.Error() {
		t.Fatalf("the fault was not recorded: %+v, %v", o, err)
	}
}

// panicker is a window whose progress events panic mid-check, standing in for
// any fault inside one; the closing event, which is empty, goes through.
type panicker struct{}

func (panicker) Emit(_ string, data any) {
	if progress, ok := data.(ProgressDTO); ok && progress.Total > 0 {
		panic("planted")
	}
}

func TestRefreshRunsACheckAndEndsWithAnEmptyProgress(t *testing.T) {
	r := newRig(t, nil)
	if _, err := r.app.SaveWebsite(0, "example.org", []string{widgetRepo.String()}); err != nil {
		t.Fatal(err)
	}
	if err := r.app.Refresh(); err != nil {
		t.Fatal(err)
	}
	waitFor(t, func() bool {
		events := r.window.seen()
		return len(events) > 1 && events[len(events)-1] == ProgressDTO{}
	})
	if first := r.window.seen()[0]; first.Total != 1 || first.Site == "" {
		t.Fatalf("first progress = %+v", first)
	}
}

// A first run starts with no websites: nothing is built into the binary for
// another owner to inherit (Amendment 12).
func TestStartupAddsNothingAndShutdownCloses(t *testing.T) {
	r := newRig(t, nil)
	closed := false
	r.app.close = func() error { closed = true; return errPlanted }
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	r.app.startup(ctx)
	waitFor(t, func() bool { return len(r.window.seen()) > 0 })
	if o, _ := r.app.Overview(); len(o.Rows) != 0 {
		t.Fatalf("a first run started with websites: %+v", o.Rows)
	}
	r.app.ready(ctx)
	r.app.focuser = nil
	r.app.ready(ctx)
	if r.window.focused != 1 {
		t.Fatalf("focused %d times, want 1", r.window.focused)
	}
	r.app.shutdown(ctx)
	if !closed {
		t.Fatal("shutdown left the data open")
	}
}

func TestNamedSortsLargestFirstThenByName(t *testing.T) {
	got := named(map[string]int{"b": 2, "a": 2, "c": 5})
	want := []NamedCountDTO{{"c", 5}, {"a", 2}, {"b", 2}}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("named = %v, want %v", got, want)
		}
	}
}

func TestNamedLeavesOutWhatHasNoDownloads(t *testing.T) {
	got := named(map[string]int{"v1.0.0": 0, "v1.1.0": 3})
	if len(got) != 1 || got[0] != (NamedCountDTO{"v1.1.0", 3}) {
		t.Fatalf("named = %v, want only v1.1.0", got)
	}
}

func TestPlatformCountsKeepTheDomainOrderAndLeaveOutNone(t *testing.T) {
	got := platformCounts(map[domain.Platform]int{
		domain.Other: 2, domain.MacOS: 0, domain.Linux: 1, domain.Windows: 4,
	})
	want := []NamedCountDTO{{string(domain.Windows), 4}, {string(domain.Linux), 1}, {string(domain.Other), 2}}
	if len(got) != len(want) {
		t.Fatalf("platformCounts = %v, want %v", got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("platformCounts = %v, want %v", got, want)
		}
	}
}

// waitFor polls cond until it holds or the test's patience runs out.
func waitFor(t *testing.T, cond func() bool) {
	t.Helper()
	const patience, step = 5 * time.Second, 10 * time.Millisecond
	for deadline := time.Now().Add(patience); time.Now().Before(deadline); time.Sleep(step) {
		if cond() {
			return
		}
	}
	t.Fatal("the condition never held")
}
