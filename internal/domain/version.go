package domain

// Comparing Visitron's own version with a published release's (FR-075),
// ported from Bridge Talk's update service.

import (
	"strconv"
	"strings"
)

// tagPrefix is the letter a release's tag may carry ahead of its version, as
// in v1.5.0.
const tagPrefix = "v"

// versionSeparator divides a version's whole numbers.
const versionSeparator = "."

// ReleaseVersion reads a release's tag as the version it names: surrounding
// space and one leading tagPrefix removed.
func ReleaseVersion(tag string) string {
	return strings.TrimPrefix(strings.TrimSpace(tag), tagPrefix)
}

// Newer reports whether latest is a later version than running. comparable is
// false when running is not dotted whole numbers, as a build from source is
// not, so no release can be said to be newer or not. A latest that cannot be
// read is never newer, so a malformed tag can never be offered.
func Newer(latest, running string) (newer, comparable bool) {
	have, ok := versionParts(running)
	if !ok {
		return false, false
	}
	offered, ok := versionParts(latest)
	if !ok {
		return false, true
	}
	return compareParts(offered, have) > 0, true
}

// compareParts answers 1 when version a is later than b, -1 when earlier and 0
// when they name the same version. The update check and the order releases
// are listed in both read versions this way, so they cannot disagree.
func compareParts(a, b []uint64) int {
	for i := range max(len(a), len(b)) {
		if l, r := partAt(a, i), partAt(b, i); l != r {
			if l > r {
				return 1
			}
			return -1
		}
	}
	return 0
}

// versionParts reads a version as dotted whole numbers; ok is false for
// anything else, a sign or a pre-release suffix included.
func versionParts(version string) ([]uint64, bool) {
	fields := strings.Split(version, versionSeparator)
	parts := make([]uint64, 0, len(fields))
	for _, field := range fields {
		number, err := strconv.ParseUint(field, 10, 64)
		if err != nil {
			return nil, false
		}
		parts = append(parts, number)
	}
	return parts, true
}

// partAt answers the version's number at index i; zero past its end, so
// 1.5.0.1 is newer than 1.5.0 while 1.5.0 is not newer than 1.5.
func partAt(parts []uint64, i int) uint64 {
	if i < len(parts) {
		return parts[i]
	}
	return 0
}
