package structural

import (
	"path/filepath"
	"regexp"
	"strings"
	"testing"
)

// inputTag matches one JSX input element, across lines. Arrow functions in its
// attributes carry "=>" but never "/>", so the shortest match ends the tag.
var inputTag = regexp.MustCompile(`(?s)<input\b.*?/>`)

// appliesAsItChanges matches the inputs that need no Enter: a tick, a choice
// or a number takes effect the moment it changes.
var appliesAsItChanges = regexp.MustCompile(`type="(checkbox|radio|number)"`)

// TestEveryTextBoxAppliesOnEnter holds Amendment 12: Enter in a text box does
// what its button does, so the owner never has to reach for the mouse. Every
// text or password box carries the shared onEnter handler from keys.ts.
//
// Proved by planting: removing onEnter from the GoatCounter site box in
// SettingsDialog.tsx fails it by name.
func TestEveryTextBoxAppliesOnEnter(t *testing.T) {
	boxes := 0
	for _, path := range frontendFiles(t) {
		if !strings.EqualFold(filepath.Ext(path), ".tsx") || strings.Contains(path, ".test.") {
			continue
		}
		for _, tag := range inputTag.FindAllString(readFile(t, path), -1) {
			if appliesAsItChanges.MatchString(tag) {
				continue
			}
			boxes++
			if !strings.Contains(tag, "onEnter(") {
				t.Errorf("%s has a text box that ignores Enter: %s", filepath.Base(path), firstLine(tag))
			}
		}
	}
	if boxes == 0 {
		t.Fatal("found no text boxes at all: the scan is wrong")
	}
}

func firstLine(text string) string {
	if i := strings.IndexByte(text, '\n'); i >= 0 {
		return text[:i]
	}
	return text
}
