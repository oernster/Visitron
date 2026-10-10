package structural

// Reading a stylesheet well enough to judge it: splitting a sheet into rules,
// telling a ring rule from any other, telling what a rule is aimed at, then
// telling a declaration that draws something from one that turns something off.
//
// It lives beside focus_test.go rather than inside it because the two are
// different concerns: this is how a sheet is read, that is what the reading
// must show.

import (
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
)

// ringStates are the pseudo-classes that make a rule a ring rule. focus-within
// counts: a ring drawn on a wrapper because something inside it took focus is
// still a ring on the wrapper.
var ringStates = []string{":focus-visible", ":focus-within", ":focus", ":hover"}

// containerTags are the elements whose whole job is to hold other things. None
// of them may carry a ring rule or a tabindex.
var containerTags = map[string]bool{
	"div": true, "section": true, "main": true, "article": true, "aside": true,
	"header": true, "footer": true, "nav": true, "form": true, "fieldset": true,
	"ul": true, "ol": true, "li": true, "table": true, "tbody": true, "tr": true,
	"p": true, "span": true, "label": true,
}

// controlTags are the elements a reader acts on; the only ones that may carry a
// tabindex.
var controlTags = map[string]bool{
	"button": true, "input": true, "textarea": true, "select": true, "a": true,
}

// invisibleValues are the ways a declaration says "draw nothing", which is the
// opposite of a ring and always allowed. `* { outline: none }` suppressing the
// engine's own indicator is the sanctioned use of the universal selector.
var invisibleValues = map[string]bool{
	"none": true, "0": true, "transparent": true, "unset": true, "initial": true, "hidden": true,
}

// borderDeclaration matches a declaration that could draw a ring.
var borderDeclaration = regexp.MustCompile(`(?m)(outline|border|border-color|border-top|border-bottom|border-left|border-right)\s*:\s*([^;}]+)`)

// pseudoSuffix strips the pseudo-classes and pseudo-elements off a selector
// part, leaving what the rule is actually aimed at.
var pseudoSuffix = regexp.MustCompile(`::?[a-z-]+(\([^)]*\))?`)

// scrollDeclaration matches a container being made to scroll.
var scrollDeclaration = regexp.MustCompile(`(?m)overflow(-y)?\s*:\s*(auto|scroll)\b`)

// setupPage is the setup program's hand-written front end. It is generated
// output as far as the other walks are concerned (it lives under dist), so it
// is named here rather than found.
var setupPage = []string{
	filepath.Join("installer", "frontend", "dist", "setup.css"),
	filepath.Join("installer", "frontend", "dist", "setup-reading.css"),
	filepath.Join("installer", "frontend", "dist", "index.html"),
	filepath.Join("installer", "frontend", "dist", "setup-routes.js"),
	filepath.Join("installer", "frontend", "dist", "setup-scroll.js"),
	filepath.Join("installer", "frontend", "dist", "setup-shell.js"),
}

// ringSurfaces returns every file the focus rules govern: the application's
// front end plus the setup program's page.
func ringSurfaces(t *testing.T) []string {
	t.Helper()
	found := frontendFiles(t)
	root := repoRoot(t)
	for _, relative := range setupPage {
		path := filepath.Join(root, relative)
		if _, err := os.Stat(path); err != nil {
			t.Fatalf("the setup page names %s, which is not there: %v", filepath.ToSlash(relative), err)
		}
		found = append(found, path)
	}
	return found
}

// readFile answers a file's contents, failing the test rather than the caller.
func readFile(t *testing.T, path string) string {
	t.Helper()
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("reading %s: %v", path, err)
	}
	return string(raw)
}

// cssRules splits a stylesheet into selector and block pairs. It is a naive
// split on the braces, which is enough because no rule here nests.
func cssRules(source string) [][2]string {
	var rules [][2]string
	for _, chunk := range strings.Split(source, "}") {
		open := strings.Index(chunk, "{")
		if open < 0 {
			continue
		}
		selector := strings.TrimSpace(stripComments(chunk[:open]))
		if selector == "" || strings.HasPrefix(selector, "@") {
			continue
		}
		rules = append(rules, [2]string{selector, chunk[open+1:]})
	}
	return rules
}

// stripComments removes block comments, so a selector quoted in prose is never
// read as a rule.
func stripComments(source string) string {
	for {
		open := strings.Index(source, "/*")
		if open < 0 {
			return source
		}
		shut := strings.Index(source[open:], "*/")
		if shut < 0 {
			return source[:open]
		}
		source = source[:open] + source[open+shut+2:]
	}
}

// drawsSomething reports whether a block sets a border or an outline that the
// reader would actually see.
func drawsSomething(block string) bool {
	for _, match := range borderDeclaration.FindAllStringSubmatch(block, -1) {
		value := strings.ToLower(strings.TrimSpace(match[2]))
		if invisibleValues[value] {
			continue
		}
		// A shorthand such as `border: 1px solid transparent` draws nothing
		// either; the colour is the last word that matters.
		if strings.Contains(value, "transparent") {
			continue
		}
		return true
	}
	return false
}

// suppressesNativeRing reports whether a block turns the engine's own focus
// indicator off.
func suppressesNativeRing(block string) bool {
	for _, match := range borderDeclaration.FindAllStringSubmatch(block, -1) {
		if match[1] != "outline" {
			continue
		}
		if invisibleValues[strings.ToLower(strings.TrimSpace(match[2]))] {
			return true
		}
	}
	return false
}

// suppressedFor reports whether a sheet turns the native ring off for a
// selector that was made to scroll.
func suppressedFor(rules [][2]string, scroller string) bool {
	for _, rule := range rules {
		if !isRingSelector(rule[0]) || !suppressesNativeRing(rule[1]) {
			continue
		}
		for _, part := range strings.Split(rule[0], ",") {
			if targetOf(part) == targetOf(scroller) {
				return true
			}
		}
	}
	return false
}

// isRingSelector reports whether a selector part is stated for a focus or hover
// state.
func isRingSelector(part string) bool {
	for _, state := range ringStates {
		if strings.Contains(part, state) {
			return true
		}
	}
	return false
}

// targetOf answers what a selector part is aimed at once its pseudo-classes are
// taken off: the last simple selector, since that is the element the rule
// paints.
func targetOf(part string) string {
	bare := strings.TrimSpace(pseudoSuffix.ReplaceAllString(part, ""))
	fields := strings.FieldsFunc(bare, func(r rune) bool {
		return r == ' ' || r == '>' || r == '+' || r == '~'
	})
	if len(fields) == 0 {
		return ""
	}
	return fields[len(fields)-1]
}

// tagHolding answers the name of the element whose opening tag an attribute
// sits in, by walking back to the nearest `<`.
func tagHolding(source string, at int) string {
	open := strings.LastIndex(source[:at], "<")
	if open < 0 {
		return "?"
	}
	rest := source[open+1:]
	end := strings.IndexFunc(rest, func(r rune) bool {
		return r == ' ' || r == '\n' || r == '\t' || r == '>' || r == '/'
	})
	if end < 0 {
		return "?"
	}
	return strings.ToLower(rest[:end])
}
