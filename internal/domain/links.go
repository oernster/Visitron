package domain

import (
	"net/url"
	"regexp"
	"strings"
)

// linkAttribute finds the value of every href and src attribute, quoted
// either way.
var linkAttribute = regexp.MustCompile(`(?i)\b(?:href|src)\s*=\s*["']([^"'#]+)`)

// SiteLinks lists the pages a crawl may follow from a page (FR-005): every
// href and src that resolves to http or https on the same host, under the
// same path as site, without a query or fragment, once each in order.
func SiteLinks(site Address, pageURL string, body string) []string {
	base, err := url.Parse(pageURL)
	if err != nil {
		return nil
	}
	seen := map[string]bool{}
	var links []string
	for _, m := range linkAttribute.FindAllStringSubmatch(body, -1) {
		ref, err := base.Parse(m[1])
		if err != nil || (ref.Scheme != "http" && ref.Scheme != "https") {
			continue
		}
		ref.RawQuery, ref.Fragment = "", ""
		target := Address{Host: strings.ToLower(ref.Hostname()), Path: ref.EscapedPath()}
		if target.Path == "" {
			target.Path = pathSep
		}
		if !site.Contains(target) {
			continue
		}
		if s := ref.String(); !seen[s] {
			seen[s] = true
			links = append(links, s)
		}
	}
	return links
}
