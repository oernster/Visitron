package structural

// A pane is chrome: it holds the controls, it is not one of them. A ring drawn
// round it marks nothing the reader can act on and costs them a keypress to
// step past, so no container may take focus or paint a border. The guards here
// hold that rule over both front ends, the application's and the setup
// program's, the latter because its page has no build step of its own and so is
// checked by nothing else.
//
// They also hold the self-reading cycle's own requirement: a dialog body that
// scrolls pins its action row beneath it and wears the cycle, so Close never
// drifts away as the words read themselves.
//
// How a stylesheet is read well enough to judge it is in cssscan_test.go.

import (
	"path/filepath"
	"regexp"
	"strings"
	"testing"
)

// tabIndexAttribute matches the attribute with its value, in JSX and in plain
// HTML alike. The value is captured because its sign is the whole question: a
// negative tabindex takes an element OUT of the tab order, which is the
// opposite of making it a stop.
var tabIndexAttribute = regexp.MustCompile(`(?i)\btabindex\s*=\s*["'{]?\s*(-?\d+)`)

// drawsModal matches a file that draws the Modal shell itself, as a whole tag
// name: <ModalClose, which Modal.tsx draws inside the shell, is not one.
var drawsModal = regexp.MustCompile(`<Modal\b`)

// itemViews names the item views that are a stop of their own, by the file
// that draws each one: the website table is ONE stop whose arrows walk its rows
// and whose selection drives the detail (keeb invariant 4). It paints no ring;
// the selected row is the indicator. A new one is declared here, which is the
// moment to ask whether it really is a list rather than a pane.
var itemViews = map[string]string{"WebsiteList.tsx": "table"}

// These are vacuity floors: a rename that stopped the scans matching would
// otherwise turn every assertion below into a silent pass over an empty list.
const (
	leastRingRules     = 3
	leastScrollingBody = 2
)

// TestNoRingRuleNamesAContainer holds the ring to the controls.
//
// Proved by planting: a sheet holding `div:hover { border: 1px solid
// var(--border) }` fails this by name.
func TestNoRingRuleNamesAContainer(t *testing.T) {
	scanned := 0
	for _, path := range ringSurfaces(t) {
		if !strings.EqualFold(filepath.Ext(path), ".css") {
			continue
		}
		for _, rule := range cssRules(readFile(t, path)) {
			if !drawsSomething(rule[1]) {
				continue
			}
			for _, part := range strings.Split(rule[0], ",") {
				part = strings.TrimSpace(part)
				if !isRingSelector(part) {
					continue
				}
				scanned++
				target := targetOf(part)
				if target == "" {
					t.Errorf("%s: %q rings whatever takes focus; name the controls",
						filepath.Base(path), part)
					continue
				}
				if target == "*" || containerTags[strings.ToLower(target)] {
					t.Errorf("%s: %q rings a container; the ring belongs to a control",
						filepath.Base(path), part)
				}
			}
		}
	}
	if scanned < leastRingRules {
		t.Fatalf("only %d ring rules were scanned, fewer than the %d that exist: the scan is wrong",
			scanned, leastRingRules)
	}
}

// TestEveryScrollingSurfaceSuppressesTheNativeRing closes the half of the rule
// that removing a stylesheet rule cannot reach.
//
// Measured in Chromium against the built stylesheet: an overflowing scroll
// container is keyboard focusable with no tabindex at all; a Tab press drew the
// engine's own `1px auto` ring round the whole of the Guide. The surface stays
// a stop, since it carries no controls of its own and a reader has to be able
// to scroll it; it must paint nothing while it is one.
//
// Proved by planting: deleting the `.repo-list:focus` suppression from
// visitron.css fails this by name.
func TestEveryScrollingSurfaceSuppressesTheNativeRing(t *testing.T) {
	for _, path := range ringSurfaces(t) {
		if !strings.EqualFold(filepath.Ext(path), ".css") {
			continue
		}
		rules := cssRules(readFile(t, path))
		// A sheet may suppress the indicator for everything at once, which
		// covers each of its scrollers without naming them.
		everywhere := false
		for _, rule := range rules {
			if targetOf(rule[0]) == "" && isRingSelector(rule[0]) && suppressesNativeRing(rule[1]) {
				everywhere = true
			}
		}
		if everywhere {
			continue
		}
		for _, rule := range rules {
			if !scrollDeclaration.MatchString(rule[1]) {
				continue
			}
			if !suppressedFor(rules, rule[0]) {
				t.Errorf("%s: %q scrolls without turning the engine's own focus ring off: Tab would ring the whole pane",
					filepath.Base(path), rule[0])
			}
		}
	}
}

// TestNoContainerTakesFocus keeps containers out of the tab order, which is the
// other half of the rule: a pane that cannot be reached cannot be ringed.
//
// A NEGATIVE tabindex must pass: it is how the window's focus sink says
// it holds focus without being a stop, then how the ring's own selector says
// which elements to leave out. Both were refused by an earlier form of this
// check that read the attribute without reading its value.
//
// Proved by planting: a component drawing `<div tabIndex={0} />` fails this
// by name.
func TestNoContainerTakesFocus(t *testing.T) {
	for _, path := range ringSurfaces(t) {
		source := readFile(t, path)
		for _, found := range tabIndexAttribute.FindAllStringSubmatchIndex(source, -1) {
			value := source[found[2]:found[3]]
			if strings.HasPrefix(value, "-") {
				continue
			}
			tag := tagHolding(source, found[0])
			if itemViews[filepath.Base(path)] == tag {
				continue
			}
			if !controlTags[tag] {
				t.Errorf("%s: <%s> carries a tabindex of %s; only a control may be a stop",
					filepath.Base(path), tag, value)
			}
		}
	}
}

// TestEveryRingedControlSaysWhenItIsInert holds the two halves of the
// three-state model together: nothing at rest, a green ring while a control is
// hovered or focused, a permanent red one while it is disabled. A control that
// says "you can use this" and never says the opposite tells the reader half a
// story; the half it leaves out is the one they cannot work out by trying.
//
// Only element-type selectors are judged. A class may name something that
// cannot be disabled at all: the setup page rings a label wrapping a hidden
// input, where `label:disabled` is not a state that exists.
//
// Proved by planting: a sheet ringing `textarea:focus-visible` with no
// `textarea:disabled` rule fails this by name.
func TestEveryRingedControlSaysWhenItIsInert(t *testing.T) {
	for _, path := range ringSurfaces(t) {
		if !strings.EqualFold(filepath.Ext(path), ".css") {
			continue
		}
		rules := cssRules(readFile(t, path))
		inert := dangerTargets(rules)
		for target := range drawnTargets(rules, ringStates...) {
			if !controlTags[target] || inert[target] {
				continue
			}
			t.Errorf("%s: <%s> rings when it is usable and says nothing when it is not: it wants a :disabled rule",
				filepath.Base(path), target)
		}
	}
}

// dangerTargets answers the element types a sheet rings in the danger colour
// while they are disabled. The colour is part of the question: a disabled rule
// that merely restates the resting border says nothing; an earlier form of this
// check accepted exactly that, so deleting the red ring passed.
func dangerTargets(rules [][2]string) map[string]bool {
	found := map[string]bool{}
	for _, rule := range rules {
		if !strings.Contains(rule[0], ":disabled") {
			continue
		}
		rings := false
		for _, match := range borderDeclaration.FindAllStringSubmatch(rule[1], -1) {
			if strings.Contains(match[2], "--danger") {
				rings = true
			}
		}
		if !rings {
			continue
		}
		for _, part := range strings.Split(rule[0], ",") {
			found[strings.ToLower(targetOf(strings.TrimSpace(part)))] = true
		}
	}
	return found
}

// drawnTargets answers the element types a sheet draws a border or an outline
// on while in any of the given states.
func drawnTargets(rules [][2]string, states ...string) map[string]bool {
	found := map[string]bool{}
	for _, rule := range rules {
		if !drawsSomething(rule[1]) {
			continue
		}
		for _, part := range strings.Split(rule[0], ",") {
			part = strings.TrimSpace(part)
			for _, state := range states {
				if strings.Contains(part, state) {
					found[strings.ToLower(targetOf(part))] = true
				}
			}
		}
	}
	return found
}

// TestEveryScrollingDialogPinsItsActionsAndReadsItself holds the two things a
// scrolling dialog body needs: the action row outside it, so Close cannot
// scroll off a short window; the self-reading cycle on it, so a new dialog
// cannot quietly ship without the cycle every other one has.
//
// Proved by planting: dropping `pinnedActions` from GuideDialog.tsx fails the
// first check by name.
func TestEveryScrollingDialogPinsItsActionsAndReadsItself(t *testing.T) {
	const (
		bodyClass = `className="dialog-body"`
		pinned    = "pinnedActions"
		cycle     = "ref={autoScroll}"
	)
	scrolling := 0
	for _, path := range frontendFiles(t) {
		if !strings.EqualFold(filepath.Ext(path), ".tsx") || strings.Contains(path, ".test.") {
			continue
		}
		source := readFile(t, path)
		// Only a file that draws a dialog is judged. The shell that implements
		// the layout declares the prop without wearing it.
		if !drawsModal.MatchString(source) {
			continue
		}
		holdsBody := strings.Contains(source, bodyClass)
		if holdsBody {
			scrolling++
		}
		if holdsBody && !strings.Contains(source, pinned) {
			t.Errorf("%s scrolls a body without pinning its actions: Close would scroll away with the words",
				filepath.Base(path))
		}
		if holdsBody && !strings.Contains(source, cycle) {
			t.Errorf("%s scrolls a body that does not read itself: it wants %s",
				filepath.Base(path), cycle)
		}
		if !holdsBody && strings.Contains(source, pinned) {
			t.Errorf("%s pins its actions with nothing to scroll: the dialog would clip its own content",
				filepath.Base(path))
		}
	}
	if scrolling < leastScrollingBody {
		t.Fatalf("found %d scrolling dialog bodies, fewer than the %d that exist: the scan is wrong",
			scrolling, leastScrollingBody)
	}
}
