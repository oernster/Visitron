package structural

import (
	"fmt"
	"math"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"testing"
)

// themeFile holds every colour Visitron uses; nothing else may name one.
var themeFile = filepath.Join("frontend", "src", "theme.css")

// minimumContrast is WCAG 2.2 AA for body text (Amendment 3). Visitron's text
// is body size throughout, so the larger-text allowance never applies.
const minimumContrast = 4.5

// pairs are the text-on-background pairings the page actually draws. Each is
// held to the minimum in both modes.
var pairs = [][2]string{
	{"text", "bg"},
	{"text", "surface"},
	{"muted", "surface"},
	{"muted", "bg"},
	{"text", "selected"},
	{"on-accent", "accent"},
	{"on-danger", "danger"},
	{"danger", "surface"},
	{"accent", "surface"},
	// A disabled control is filled with --muted-fill and labelled in --muted,
	// so that pairing is read as often as any other.
	{"muted", "muted-fill"},
}

// minimumIndicatorContrast is what WCAG 2.2 asks of a non-text indicator
// (1.4.11). A ring is a border rather than a letter, so it is held to 3:1
// rather than to the body-text figure above.
const minimumIndicatorContrast = 3.0

// indicators are the ring colours against the surfaces they are drawn on. The
// ring says "you can use this" and the danger ring says "you cannot"; either
// one too faint to see is the same as not being drawn.
var indicators = [][2]string{
	{"ring", "surface"},
	{"ring", "bg"},
	{"danger", "muted-fill"},
}

// hexColour matches a token declaration such as `--text: #1b1f24;`.
var hexColour = regexp.MustCompile(`--([a-z-]+):\s*(#[0-9a-fA-F]{6})`)

// lightMarker splits the theme into its two blocks. Visitron opens dark, so the
// dark tokens are the plain `:root` ones and everything from this selector on
// is the light override (FR-070).
const lightMarker = ":root[data-theme='light']"

// channel converts one 8-bit component to its linear value.
func channel(value float64) float64 {
	value /= 255
	if value <= 0.04045 {
		return value / 12.92
	}
	return math.Pow((value+0.055)/1.055, 2.4)
}

// luminance answers a colour's relative luminance, as WCAG defines it.
func luminance(hex string) (float64, error) {
	value, err := strconv.ParseUint(strings.TrimPrefix(hex, "#"), 16, 32)
	if err != nil {
		return 0, fmt.Errorf("unreadable colour %q: %w", hex, err)
	}
	red := channel(float64((value >> 16) & 0xff))
	green := channel(float64((value >> 8) & 0xff))
	blue := channel(float64(value & 0xff))
	return 0.2126*red + 0.7152*green + 0.0722*blue, nil
}

// contrast answers the WCAG contrast ratio between two colours.
func contrast(t *testing.T, first, second string) float64 {
	t.Helper()
	one, err := luminance(first)
	if err != nil {
		t.Fatal(err)
	}
	other, err := luminance(second)
	if err != nil {
		t.Fatal(err)
	}
	lighter, darker := math.Max(one, other), math.Min(one, other)
	return (lighter + 0.05) / (darker + 0.05)
}

// tokensIn reads the colour tokens out of a slice of the theme.
func tokensIn(source string) map[string]string {
	found := map[string]string{}
	for _, match := range hexColour.FindAllStringSubmatch(source, -1) {
		found[match[1]] = match[2]
	}
	return found
}

// themeModes reads the theme and answers each mode's tokens. The dark block
// opens at the media query, so everything before it is light.
func themeModes(t *testing.T) map[string]map[string]string {
	t.Helper()
	raw, err := os.ReadFile(filepath.Join(repoRoot(t), themeFile))
	if err != nil {
		t.Fatalf("reading %s: %v", filepath.ToSlash(themeFile), err)
	}
	theme := string(raw)
	split := strings.Index(theme, lightMarker)
	if split < 0 {
		t.Fatalf("%s carries no light mode", filepath.ToSlash(themeFile))
	}
	return map[string]map[string]string{
		"dark":  tokensIn(theme[:split]),
		"light": tokensIn(theme[split:]),
	}
}

// TestEveryTextPairingMeetsAA holds the palette to WCAG 2.2 AA in both modes.
//
// Proved by planting a failing shade: setting the light --muted to #8d97a3
// fails this by name.
func TestEveryTextPairingMeetsAA(t *testing.T) {
	for mode, tokens := range themeModes(t) {
		for _, pair := range pairs {
			front, back := tokens[pair[0]], tokens[pair[1]]
			if front == "" || back == "" {
				t.Errorf("%s mode names no --%s or no --%s", mode, pair[0], pair[1])
				continue
			}
			if ratio := contrast(t, front, back); ratio < minimumContrast {
				t.Errorf("%s mode: --%s on --%s is %.2f:1, below %.1f:1",
					mode, pair[0], pair[1], ratio, minimumContrast)
			}
		}
	}
}

// TestEveryRingIsVisibleAgainstWhatItIsDrawnOn holds the three-state focus
// model's two ring colours to the contrast WCAG asks of an indicator, in both
// modes. The greens differ between the modes on purpose: the pastel that reads
// on near-black is weak on white.
//
// Proved by planting: setting the light --ring to #34d399, the dark mode's
// green, fails this by name.
func TestEveryRingIsVisibleAgainstWhatItIsDrawnOn(t *testing.T) {
	for mode, tokens := range themeModes(t) {
		for _, pair := range indicators {
			front, back := tokens[pair[0]], tokens[pair[1]]
			if front == "" || back == "" {
				t.Errorf("%s mode names no --%s or no --%s", mode, pair[0], pair[1])
				continue
			}
			if ratio := contrast(t, front, back); ratio < minimumIndicatorContrast {
				t.Errorf("%s mode: --%s on --%s is %.2f:1, below %.1f:1",
					mode, pair[0], pair[1], ratio, minimumIndicatorContrast)
			}
		}
	}
}

// TestNoColourIsNamedOutsideTheTheme keeps one home for every colour value.
// The print rules are the exception: paper is white and ink is black whatever
// the screen is doing, so those two are stated where they apply.
func TestNoColourIsNamedOutsideTheTheme(t *testing.T) {
	allowed := map[string]bool{"#ffffff": true, "#000000": true}
	for _, path := range frontendFiles(t) {
		if strings.HasSuffix(path, "theme.css") {
			continue
		}
		raw, err := os.ReadFile(path)
		if err != nil {
			t.Fatalf("reading %s: %v", path, err)
		}
		for _, found := range regexp.MustCompile(`#[0-9a-fA-F]{3,8}\b`).FindAllString(string(raw), -1) {
			if !allowed[strings.ToLower(found)] {
				t.Errorf("%s names the colour %s: colours live in %s",
					filepath.Base(path), found, filepath.ToSlash(themeFile))
			}
		}
	}
}
