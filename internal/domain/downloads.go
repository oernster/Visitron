package domain

import "strings"

// Platform is the operating system a release file is for.
type Platform string

// The platforms, told apart by file extension (Appendix A, M-5).
const (
	Windows Platform = "Windows"
	MacOS   Platform = "macOS"
	Linux   Platform = "Linux"
	Other   Platform = "Other"
)

const dmgExt = ".dmg"

var platformByExt = map[string]Platform{
	".exe":     Windows,
	dmgExt:     MacOS,
	".flatpak": Linux,
}

// Platforms lists every platform in the order the window shows them.
var Platforms = []Platform{Windows, MacOS, Linux, Other}

// PlatformOf names the platform of a release file by its extension, ignoring
// case; anything unrecognised is Other.
func PlatformOf(fileName string) Platform {
	lower := strings.ToLower(fileName)
	for ext, p := range platformByExt {
		if strings.HasSuffix(lower, ext) {
			return p
		}
	}
	return Other
}

// Counted is a release file's downloads less selfDownloads when it is a .dmg,
// never below zero (FR-021, Amendment 19). selfDownloads is the owner's own
// downloads of each disk image, as set in Settings; a value below none is read
// as none, so a damaged setting can never add downloads.
func Counted(fileName string, raw, selfDownloads int) int {
	if strings.HasSuffix(strings.ToLower(fileName), dmgExt) {
		raw -= max(selfDownloads, MinSelfDownloads)
	}
	return max(raw, 0)
}

// ReleaseFile is one asset of one release with GitHub's download count.
type ReleaseFile struct {
	Repo    Repo
	Release string
	Name    string
	Raw     int
}

// Counted is this file's counted downloads, less selfDownloads on a .dmg.
func (f ReleaseFile) Counted(selfDownloads int) int { return Counted(f.Name, f.Raw, selfDownloads) }

// Platform is this file's platform.
func (f ReleaseFile) Platform() Platform { return PlatformOf(f.Name) }

// FileKey identifies a release file across snapshots.
func (f ReleaseFile) FileKey() string {
	return strings.ToLower(f.Repo.String()) + pathSep + f.Release + pathSep + f.Name
}

// Totals are counted downloads summed four ways (FR-022).
type Totals struct {
	All        int
	ByRepo     map[string]int
	ByRelease  map[string]int
	ByPlatform map[Platform]int
}

// Total sums the counted downloads of files, less selfDownloads on each .dmg.
// Releases are keyed owner/name then the release tag, so two repos with the
// same tag stay apart.
func Total(files []ReleaseFile, selfDownloads int) Totals {
	t := Totals{
		ByRepo:     map[string]int{},
		ByRelease:  map[string]int{},
		ByPlatform: map[Platform]int{},
	}
	for _, f := range files {
		n := f.Counted(selfDownloads)
		t.All += n
		t.ByRepo[f.Repo.String()] += n
		t.ByRelease[f.Repo.String()+pathSep+f.Release] += n
		t.ByPlatform[f.Platform()] += n
	}
	return t
}
