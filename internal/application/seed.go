package application

import (
	"encoding/json"
	"fmt"

	"github.com/oernster/visitron/internal/domain"
)

// seedFile is the shape of sites.seed.json.
type seedFile struct {
	Sites []struct {
		URL   string   `json:"url"`
		Repos []string `json:"repos"`
	} `json:"sites"`
}

// ParseSeed reads the seed file into the websites it lists (FR-011). The file
// is embedded in the binary, so a fault in it is a build defect: it is
// refused whole rather than half used.
func ParseSeed(data []byte) ([]Website, error) {
	var f seedFile
	if err := json.Unmarshal(data, &f); err != nil {
		return nil, fmt.Errorf("the seed file: %w", err)
	}
	sites := make([]Website, 0, len(f.Sites))
	for _, s := range f.Sites {
		addr, err := domain.Normalise(s.URL)
		if err != nil {
			return nil, fmt.Errorf("the seed file, %q: %w", s.URL, err)
		}
		w := Website{Address: addr}
		for _, text := range s.Repos {
			repo, err := domain.ParseRepo(text)
			if err != nil {
				return nil, fmt.Errorf("the seed file, %q: %w", text, err)
			}
			w.Repos = append(w.Repos, repo)
		}
		sites = append(sites, w)
	}
	return sites, nil
}
