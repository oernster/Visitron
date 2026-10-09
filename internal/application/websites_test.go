package application

import (
	"context"
	"errors"
	"fmt"
	"reflect"
	"testing"

	"github.com/oernster/visitron/internal/domain"
)

var symRepo = domain.Repo{Owner: "oernster", Name: "SymDiary"}

// symdiarySite is symdiary.com as measured (Appendix A, M-3): the repo named
// on the home page and through the API in site.js, the buttons on a second
// page.
func symdiarySite() *fakeFetcher {
	return &fakeFetcher{pages: map[string]string{
		"https://symdiary.com/": `<script src="site.js?v=1"></script>
			<a href="download.html">Get</a> <a href="https://github.com/oernster/SymDiary">Code</a>`,
		"https://symdiary.com/site.js":       `fetch('https://api.github.com/repos/oernster/SymDiary/releases/latest')`,
		"https://symdiary.com/download.html": `<a href="https://github.com/oernster/SymDiary/releases/latest/download/SymDiary.dmg">dmg</a>`,
	}}
}

func TestAddCrawlsAndOffers(t *testing.T) {
	t.Parallel()
	fetch := symdiarySite()
	svc := NewWebsites(newStore(), fetch, &fakeReleases{})
	p, err := svc.Propose(context.Background(), "SymDiary.com", 0)
	if err != nil {
		t.Fatal(err)
	}
	if p.Address.URL() != "https://symdiary.com/" || p.Problem != "" {
		t.Errorf("proposal = %+v", p)
	}
	want := []domain.Repo{symRepo}
	if !reflect.DeepEqual(p.Found, want) || !reflect.DeepEqual(p.Ticked, want) {
		t.Errorf("found %v ticked %v; want %v", p.Found, p.Ticked, want)
	}
	if len(fetch.fetched) != 3 {
		t.Errorf("fetched %v; want the three pages once each", fetch.fetched)
	}
}

func TestProposeRefusesBadEntry(t *testing.T) {
	t.Parallel()
	svc := NewWebsites(newStore(), &fakeFetcher{}, &fakeReleases{})
	if _, err := svc.Propose(context.Background(), "symdiary", 0); !errors.Is(err, domain.ErrHostNotDNS) {
		t.Errorf("err = %v", err)
	}
	store := newStore()
	store.failOn = "Websites"
	if _, err := NewWebsites(store, &fakeFetcher{}, &fakeReleases{}).Propose(context.Background(), "a.com", 0); !errors.Is(err, errPlanted) {
		t.Errorf("store failure = %v", err)
	}
}

func TestDuplicateRefused(t *testing.T) {
	t.Parallel()
	store := newStore()
	addr, _ := domain.Normalise("symdiary.com")
	id, _ := store.AddWebsite(Website{Address: addr})
	svc := NewWebsites(store, symdiarySite(), &fakeReleases{})
	if _, err := svc.Propose(context.Background(), "https://SymDiary.com/", 0); !errors.Is(err, ErrDuplicate) {
		t.Errorf("err = %v; want ErrDuplicate", err)
	}
	if _, err := svc.Propose(context.Background(), "symdiary.com", id); err != nil {
		t.Errorf("editing a website to its own address was refused: %v", err)
	}
}

func TestEditRecrawlsKeepingTicks(t *testing.T) {
	t.Parallel()
	store := newStore()
	addr, _ := domain.Normalise("old.example.com")
	extra := domain.Repo{Owner: "oernster", Name: "Extra"}
	id, _ := store.AddWebsite(Website{Address: addr, Repos: []domain.Repo{extra}})
	fetch := &fakeFetcher{pages: map[string]string{
		"https://hub.example.com/": `github.com/oernster/Extra github.com/oernster/Other github.com/oernster/Third`,
	}}
	p, err := NewWebsites(store, fetch, &fakeReleases{}).Propose(context.Background(), "hub.example.com", id)
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(p.Ticked, []domain.Repo{extra}) || len(p.Found) != 3 {
		t.Errorf("ticked %v found %v", p.Ticked, p.Found)
	}
}

func TestUnreachableSiteCanBeSaved(t *testing.T) {
	t.Parallel()
	store := newStore()
	svc := NewWebsites(store, &fakeFetcher{}, &fakeReleases{})
	p, err := svc.Propose(context.Background(), "gone.example.com", 0)
	if err != nil || p.Problem == "" || len(p.Found) != 0 {
		t.Fatalf("proposal = %+v, %v; want a stated problem", p, err)
	}
	id, err := svc.Save(0, p.Address, []domain.Repo{symRepo})
	if err != nil || id == 0 || len(store.sites) != 1 {
		t.Errorf("save after a failed crawl: id %d err %v", id, err)
	}
}

func TestBrokenInnerLinkIsSkipped(t *testing.T) {
	t.Parallel()
	fetch := &fakeFetcher{pages: map[string]string{
		"https://a.example.com/": `<a href="missing.html">x</a> github.com/oernster/A`,
	}}
	p, _ := NewWebsites(newStore(), fetch, &fakeReleases{}).Propose(context.Background(), "a.example.com", 0)
	if p.Problem != "" || len(p.Found) != 1 {
		t.Errorf("proposal = %+v", p)
	}
}

func TestCrawlStopsAtPageLimit(t *testing.T) {
	t.Parallel()
	pages := map[string]string{}
	for i := 0; i <= CrawlPageLimit; i++ {
		pages[fmt.Sprintf("https://big.example.com/%d.html", i)] = fmt.Sprintf(`<a href="%d.html">next</a>`, i+1)
	}
	pages["https://big.example.com/"] = `<a href="0.html">start</a>`
	fetch := &fakeFetcher{pages: pages}
	_, _ = NewWebsites(newStore(), fetch, &fakeReleases{}).Propose(context.Background(), "big.example.com", 0)
	if len(fetch.fetched) != CrawlPageLimit {
		t.Errorf("fetched %d pages; the limit is %d", len(fetch.fetched), CrawlPageLimit)
	}
}

func TestManualRepository(t *testing.T) {
	t.Parallel()
	rel := &fakeReleases{files: map[string][]domain.ReleaseFile{"oernster/SymDiary": nil}}
	svc := NewWebsites(newStore(), &fakeFetcher{}, rel)
	if r, err := svc.Confirm(context.Background(), "oernster/SymDiary"); err != nil || r != symRepo {
		t.Errorf("Confirm = %v, %v", r, err)
	}
	if _, err := svc.Confirm(context.Background(), "oernster/Nope"); !errors.Is(err, ErrRepoNotFound) {
		t.Errorf("missing repo = %v", err)
	}
	if _, err := svc.Confirm(context.Background(), "nope"); !errors.Is(err, domain.ErrRepoForm) {
		t.Errorf("bad form = %v", err)
	}
	rel.existsErr = errPlanted
	if _, err := svc.Confirm(context.Background(), "oernster/SymDiary"); !errors.Is(err, errPlanted) {
		t.Errorf("GitHub failure = %v", err)
	}
}

func TestSaveUpdatesAndDeleteRemoves(t *testing.T) {
	t.Parallel()
	store := newStore()
	svc := NewWebsites(store, &fakeFetcher{}, &fakeReleases{})
	a, _ := domain.Normalise("a.example.com")
	b, _ := domain.Normalise("b.example.com")
	id, _ := svc.Save(0, a, nil)
	if _, err := svc.Save(id, b, []domain.Repo{symRepo}); err != nil {
		t.Fatal(err)
	}
	list, _ := svc.List()
	if len(list) != 1 || list[0].Address != b || len(list[0].Repos) != 1 {
		t.Errorf("after update: %+v", list)
	}
	if err := svc.Delete(id); err != nil || len(store.sites) != 0 {
		t.Errorf("delete: %v, %d left", err, len(store.sites))
	}
}

func TestSeedsOnceOnly(t *testing.T) {
	t.Parallel()
	store := newStore()
	svc := NewWebsites(store, &fakeFetcher{}, &fakeReleases{})
	a, _ := domain.Normalise("a.example.com")
	seed := []Website{{Address: a}}
	if did, err := svc.Seed(seed); !did || err != nil || len(store.sites) != 1 {
		t.Fatalf("first run: %v %v %d", did, err, len(store.sites))
	}
	_ = svc.Delete(store.sites[0].ID)
	if did, _ := svc.Seed(seed); did || len(store.sites) != 0 {
		t.Error("seeded again after the owner deleted everything")
	}
}

func TestSeedSkipsWhenSitesExist(t *testing.T) {
	t.Parallel()
	store := newStore()
	a, _ := domain.Normalise("a.example.com")
	_, _ = store.AddWebsite(Website{Address: a})
	did, err := NewWebsites(store, &fakeFetcher{}, &fakeReleases{}).Seed([]Website{{Address: a}, {Address: a}})
	if did || err != nil || len(store.sites) != 1 || !store.seeded {
		t.Errorf("did %v err %v sites %d seeded %v", did, err, len(store.sites), store.seeded)
	}
}

func TestSeedFailures(t *testing.T) {
	t.Parallel()
	for _, op := range []string{"Seeded", "Websites", "AddWebsite"} {
		store := newStore()
		store.failOn = op
		a, _ := domain.Normalise("a.example.com")
		if _, err := NewWebsites(store, &fakeFetcher{}, &fakeReleases{}).Seed([]Website{{Address: a}}); !errors.Is(err, errPlanted) {
			t.Errorf("%s failure = %v", op, err)
		}
	}
}
