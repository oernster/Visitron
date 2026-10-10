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
