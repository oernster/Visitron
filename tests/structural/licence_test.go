package structural

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/oernster/visitron/internal/licence"
)

// rootLicence is the licence as the repository publishes it, which is the file
// GitHub reads and the one a person finds beside the source.
var rootLicence = "LICENSE"

// TestTheEmbeddedLicenceIsThePublishedOne keeps the one copy honest.
//
// Go's embedding cannot reach above the package it is written in, so the setup
// program cannot embed the repository's own LICENSE and a second copy lives in
// internal/licence. A second copy of a legal text is worth having only while
// it cannot drift: a setup program telling somebody they are covered by terms
// that are not the terms they are covered by is worse than one that says
// nothing. This is the whole guard, so it compares every byte rather than a
// length, a first line or a hash of either.
func TestTheEmbeddedLicenceIsThePublishedOne(t *testing.T) {
	t.Parallel()
	published, err := os.ReadFile(filepath.Join(repoRoot(t), rootLicence))
	if err != nil {
		t.Fatalf("reading %s: %v", rootLicence, err)
	}
	// Line endings are the one difference allowed: git hands a Windows
	// checkout CRLF, while the embedded copy is read as it sits on disk. The
	// words are what matters and Text normalises them, so both sides are
	// compared in the same form.
	want := normalised(string(published))
	got := licence.Text()
	if got == want {
		return
	}
	t.Errorf("the licence embedded in internal/licence is not %s: %d bytes against %d. "+
		"A person installing Visitron reads the embedded copy, so the two are the "+
		"same text or the setup program is stating terms nobody granted. Copy "+
		"%s over internal/licence/LICENSE.", rootLicence, len(got), len(want), rootLicence)
	if where := firstDifference(got, want); where >= 0 {
		t.Errorf("they first differ at byte %d", where)
	}
}

// normalised puts a text into the one line-ending form both sides are read in.
func normalised(text string) string {
	out := make([]byte, 0, len(text))
	for index := 0; index < len(text); index++ {
		if text[index] == '\r' && index+1 < len(text) && text[index+1] == '\n' {
			continue
		}
		out = append(out, text[index])
	}
	return string(out)
}

// firstDifference answers the index the two texts part company at, else -1.
// The byte is worth reporting: a licence that differs at the very end is a
// truncated copy, while one that differs early is a different document.
func firstDifference(a, b string) int {
	shorter := min(len(a), len(b))
	for index := range shorter {
		if a[index] != b[index] {
			return index
		}
	}
	if len(a) != len(b) {
		return shorter
	}
	return -1
}
