package application

import (
	"errors"
	"os"
	"strings"
	"testing"

	"github.com/oernster/visitron/internal/domain"
)

func TestTheShippedSeedParses(t *testing.T) {
	t.Parallel()
	data, err := os.ReadFile("../../sites.seed.json")
	if err != nil {
		t.Fatal(err)
	}
	sites, err := ParseSeed(data)
	if err != nil {
		t.Fatal(err)
	}
	const measured = 32 // Appendix A, M-4
	if len(sites) != measured {
		t.Errorf("seed holds %d sites; %d were measured", len(sites), measured)
	}
	if sites[0].Address.Key() != "ernster.dev/" || len(sites[0].Repos) != 0 {
		t.Errorf("first site %+v; want ernster.dev with no repo", sites[0])
	}
}

func TestSeedRefusesFaults(t *testing.T) {
	t.Parallel()
	cases := map[string]error{
		`{`:                                      nil,
		`{"sites":[{"url":"nodot","repos":[]}]}`: domain.ErrHostNotDNS,
		`{"sites":[{"url":"a.com","repos":["x"]}]}`: domain.ErrRepoForm,
	}
	for text, want := range cases {
		_, err := ParseSeed([]byte(text))
		if err == nil || !strings.Contains(err.Error(), "seed file") || (want != nil && !errors.Is(err, want)) {
			t.Errorf("ParseSeed(%s) = %v", text, err)
		}
	}
}
