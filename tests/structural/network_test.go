package structural

import (
	"go/ast"
	"go/parser"
	"go/token"
	"sort"
	"strconv"
	"strings"
	"testing"
)

// NFR-PRIV-001: Visitron contacts only GoatCounter, GitHub, the websites it
// crawls and the donate link when pressed. The two tests here are the
// structural test listing every outbound host that the requirement names.

// httpPackages are the only packages allowed to import net/http, each with why.
// A new one fails here, which is the moment to ask what it talks to.
var httpPackages = map[string]string{
	// The one HTTP client: every request Visitron makes goes through it.
	"internal/infrastructure/web": "the client",
	// Reads GitHub's status codes off the shared client's answers; it opens
	// nothing of its own.
	"internal/infrastructure/github": "status codes only",
	// Reads the update check's 404, GitHub's word for no release published;
	// its request rides the shared client.
	"internal/infrastructure/update": "status codes only",
}

// outboundAddresses are the only web addresses the Go source may name, each
// with the requirement that allows it. A crawled website is not among them:
// the owner types it, so it never appears in the source.
var outboundAddresses = map[string]string{
	"https://api.github.com":                           "GitHub's API (FR-021)",
	"https://oernster.goatcounter.com":                 "GoatCounter's API (FR-020)",
	"https://www.paypal.com/ncp/payment/NRXS4SP24A6C8": "the donate link, opened in the browser (FR-073)",
	// The only host an update's page or file may name; anything else in
	// GitHub's answer is refused before it can reach the browser.
	"https://github.com/": "a release of Visitron, opened in the browser (FR-075)",
	// The scheme the domain puts on an address typed without one.
	"https://": "the default scheme (FR-001)",
}

// TestOnlyTheListedPackagesSpeakHTTP holds every connection to the packages
// that are known to make one. A raw socket is refused outright.
func TestOnlyTheListedPackagesSpeakHTTP(t *testing.T) {
	root := repoRoot(t)
	for _, path := range goFiles(t) {
		if strings.HasSuffix(path, "_test.go") {
			continue
		}
		dir := relativeTo(root, path)
		dir = dir[:max(strings.LastIndex(dir, "/"), 0)]
		for _, imported := range importsOf(t, path) {
			if imported == "net" {
				t.Errorf("%s imports net: Visitron opens no raw connection", relativeTo(root, path))
			}
			if imported == "net/http" {
				if _, ok := httpPackages[dir]; !ok {
					t.Errorf("%s imports net/http and is not declared in httpPackages: "+
						"NFR-PRIV-001 holds every connection to known places", relativeTo(root, path))
				}
			}
		}
	}
}

// TestEveryOutboundAddressIsListed reads every string literal in the Go
// source and refuses a web address the requirement does not allow.
func TestEveryOutboundAddressIsListed(t *testing.T) {
	root := repoRoot(t)
	seen := map[string]bool{}
	for _, path := range goFiles(t) {
		if strings.HasSuffix(path, "_test.go") {
			continue
		}
		parsed, err := parser.ParseFile(token.NewFileSet(), path, nil, 0)
		if err != nil {
			t.Fatalf("parsing %s: %v", path, err)
		}
		ast.Inspect(parsed, func(node ast.Node) bool {
			literal, ok := node.(*ast.BasicLit)
			if !ok || literal.Kind != token.STRING {
				return true
			}
			text, err := strconv.Unquote(literal.Value)
			if err != nil || !(strings.HasPrefix(text, "http://") || strings.HasPrefix(text, "https://")) {
				return true
			}
			seen[text] = true
			if _, ok := outboundAddresses[text]; !ok {
				t.Errorf("%s names %q, which NFR-PRIV-001 does not list: declare it in "+
					"outboundAddresses with the requirement that allows it", relativeTo(root, path), text)
			}
			return true
		})
	}
	var stale []string
	for address := range outboundAddresses {
		if !seen[address] {
			stale = append(stale, address)
		}
	}
	sort.Strings(stale)
	for _, address := range stale {
		t.Errorf("outboundAddresses lists %q, which the source no longer names", address)
	}
}
