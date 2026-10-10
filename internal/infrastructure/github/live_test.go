package github

import (
	"context"
	"os"
	"testing"

	"visitron/internal/domain"
	"visitron/internal/infrastructure/web"
)

// liveRepo names a real public repository with releases, as owner/name, for
// the tests that reach the real GitHub. The gate never sets it. No repository
// is named here: whoever runs it names their own.
const liveRepo = "VISITRON_LIVE_REPO"

func TestLiveRepository(t *testing.T) {
	text := os.Getenv(liveRepo)
	if text == "" {
		t.Skip("set " + liveRepo + "=owner/name to read that repository on the real GitHub")
	}
	repo, err := domain.ParseRepo(text)
	if err != nil {
		t.Fatalf("%s: %v", liveRepo, err)
	}
	c := NewClient(web.NewClient(), secrets{}, DefaultBase)
	files, _, err := c.Files(context.Background(), repo)
	if err != nil || len(files) == 0 {
		t.Fatalf("files %d err %v", len(files), err)
	}
	for _, f := range files {
		t.Logf("%s %s raw %d counted %d %s", f.Release, f.Name, f.Raw, f.Counted(domain.DefaultSelfDownloads), f.Platform())
	}
	missing := domain.Repo{Owner: repo.Owner, Name: "no-such-repo-xyz"}
	if ok, err := c.Exists(context.Background(), missing); ok || err != nil {
		t.Errorf("missing repo: %v %v", ok, err)
	}
}
