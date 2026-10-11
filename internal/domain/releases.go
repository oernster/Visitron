package domain

import "strings"

// releaseKeyParts is how many pieces a release key splits into: owner, name
// and tag, as Total keys ByRelease.
const releaseKeyParts = 3

// ReleaseBefore reports whether release key a is listed before b. Keys are
// owner/name/tag, as Total builds them. Repositories go in name order; within
// one, the newest version comes first, read as numbers so 1.10.1 is newer
// than 1.9 (Amendment 24). A tag that is not a version follows every one that
// is; ties are settled by the tag's text, so the order never wavers.
func ReleaseBefore(a, b string) bool {
	repoA, tagA := splitReleaseKey(a)
	repoB, tagB := splitReleaseKey(b)
	if repoA != repoB {
		return repoA < repoB
	}
	partsA, okA := versionParts(ReleaseVersion(tagA))
	partsB, okB := versionParts(ReleaseVersion(tagB))
	switch {
	case okA != okB:
		return okA
	case okA:
		if order := compareParts(partsA, partsB); order != 0 {
			return order > 0
		}
	}
	return tagA < tagB
}

// ReleaseLabels answers how each release key reads in a table, in the same
// order. Where every release comes from one repository the tag alone says it.
// Where they span several, two of them can share a tag, so each tag carries
// its repository's name; the owner joins the name only where two of those
// repositories share a name. A key with no tag reads as itself.
func ReleaseLabels(keys []string) []string {
	repos := map[string]bool{}
	ownersOf := map[string]map[string]bool{}
	for _, key := range keys {
		parts := strings.SplitN(key, pathSep, releaseKeyParts)
		if len(parts) < releaseKeyParts {
			continue
		}
		repo, _ := splitReleaseKey(key)
		repos[repo] = true
		name := strings.ToLower(parts[1])
		if ownersOf[name] == nil {
			ownersOf[name] = map[string]bool{}
		}
		ownersOf[name][strings.ToLower(parts[0])] = true
	}
	labels := make([]string, len(keys))
	for i, key := range keys {
		parts := strings.SplitN(key, pathSep, releaseKeyParts)
		switch {
		case len(parts) < releaseKeyParts:
			labels[i] = key
		case len(repos) == 1:
			labels[i] = parts[2]
		case len(ownersOf[strings.ToLower(parts[1])]) == 1:
			labels[i] = parts[1] + " " + parts[2]
		default:
			labels[i] = parts[0] + pathSep + parts[1] + " " + parts[2]
		}
	}
	return labels
}

// splitReleaseKey answers a release key's repository with its tag. The
// repository is in lower case so case alone never parts two keys of one
// repository; the tag is "" when the key carries none.
func splitReleaseKey(key string) (repo, tag string) {
	parts := strings.SplitN(key, pathSep, releaseKeyParts)
	if len(parts) < releaseKeyParts {
		return strings.ToLower(key), ""
	}
	return strings.ToLower(parts[0] + pathSep + parts[1]), parts[2]
}
