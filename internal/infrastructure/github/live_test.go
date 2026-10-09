package github

import (
	"context"
	"os"
	"testing"

	"github.com/oernster/visitron/internal/domain"
	"github.com/oernster/visitron/internal/infrastructure/web"
)

// liveFlag runs the tests that reach the real GitHub; the gate never sets it.
const liveFlag = "VISITRON_LIVE"

func TestLiveSymDiary(t *testing.T) {
	if os.Getenv(liveFlag) == "" {
		t.Skip("set " + liveFlag + "=1 to read the real GitHub")
	}
	c := NewClient(web.NewClient(), secrets{}, DefaultBase)
	files, _, err := c.Files(context.Background(), sym)
	if err != nil || len(files) == 0 {
		t.Fatalf("files %d err %v", len(files), err)
	}
	for _, f := range files {
		t.Logf("%s %s raw %d counted %d %s", f.Release, f.Name, f.Raw, f.Counted(), f.Platform())
	}
	if ok, err := c.Exists(context.Background(), domain.Repo{Owner: "oernster", Name: "no-such-repo-xyz"}); ok || err != nil {
		t.Errorf("missing repo: %v %v", ok, err)
	}
}
