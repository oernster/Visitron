package domain

import (
	"fmt"
	"strings"
)

// Day is a calendar day in the time zone Windows is set to. The application
// layer works it out from its clock; the domain only compares and prints it.
type Day struct {
	Year  int
	Month int
	Date  int
}

// Before reports whether d comes earlier than other.
func (d Day) Before(other Day) bool {
	if d.Year != other.Year {
		return d.Year < other.Year
	}
	if d.Month != other.Month {
		return d.Month < other.Month
	}
	return d.Date < other.Date
}

// String is the ISO form, 2026-10-09.
func (d Day) String() string { return fmt.Sprintf("%04d-%02d-%02d", d.Year, d.Month, d.Date) }

// Snapshot is the counted downloads of every release file at one check, by
// file key.
type Snapshot struct {
	Day    Day
	Counts map[string]int
}

// DayFiles is the release files of one repo as read at a check on Day, with
// GitHub's own counts; nothing is taken off them until they are counted.
type DayFiles struct {
	Day   Day
	Files []ReleaseFile
}

// SnapshotOf takes the counted downloads of files as a snapshot on day, less
// selfDownloads on each .dmg.
func SnapshotOf(day Day, files []ReleaseFile, selfDownloads int) Snapshot {
	counts := make(map[string]int, len(files))
	for _, f := range files {
		counts[f.FileKey()] = f.Counted(selfDownloads)
	}
	return Snapshot{Day: day, Counts: counts}
}

// Rise is the downloads between two kept snapshots (FR-023): for each file
// present in both, the amount its count went up. A file whose count fell adds
// nothing, nor does a file missing from either (FR-024), because a deleted or
// re-uploaded file restarts GitHub's count.
func Rise(earlier, later Snapshot) int {
	total := 0
	for key, now := range later.Counts {
		if before, ok := earlier.Counts[key]; ok && now > before {
			total += now - before
		}
	}
	return total
}

// LatestRiseBetween is the rise in counted downloads of each repo's latest
// release from one reading of release files to a later one, less
// selfDownloads on each .dmg (Amendment 34). The latest release is the newest
// version in the later reading, as ReleaseBefore orders them. Its files that
// the earlier reading lacks were released between the two, so they count from
// none; as with Rise, a file that fell adds nothing. With no earlier reading
// there is nothing to rise from, so it answers none.
func LatestRiseBetween(earlier, later []ReleaseFile, selfDownloads int) int {
	if len(earlier) == 0 {
		return 0
	}
	newest := map[string]string{}
	for _, f := range later {
		repo := strings.ToLower(f.Repo.String())
		if key, ok := newest[repo]; !ok || ReleaseBefore(f.ReleaseKey(), key) {
			newest[repo] = f.ReleaseKey()
		}
	}
	var latest []ReleaseFile
	for _, f := range later {
		if newest[strings.ToLower(f.Repo.String())] == f.ReleaseKey() {
			latest = append(latest, f)
		}
	}
	before := SnapshotOf(Day{}, earlier, selfDownloads)
	for _, f := range latest {
		if _, ok := before.Counts[f.FileKey()]; !ok {
			before.Counts[f.FileKey()] = 0
		}
	}
	return Rise(before, SnapshotOf(Day{}, latest, selfDownloads))
}

// DailyRises keeps the last snapshot of each day in day order, then reports
// each day's rise over the day kept before it. The first day kept has nothing
// to rise from, so it reports none.
func DailyRises(snapshots []Snapshot) []DayCount {
	kept := lastPerDay(snapshots)
	rises := make([]DayCount, 0, len(kept))
	for i := 1; i < len(kept); i++ {
		rises = append(rises, DayCount{Day: kept[i].Day, Count: Rise(kept[i-1], kept[i])})
	}
	return rises
}

// DayCount is one day's figure.
type DayCount struct {
	Day   Day
	Count int
}

func lastPerDay(snapshots []Snapshot) []Snapshot {
	var kept []Snapshot
	for _, s := range snapshots {
		i := 0
		for i < len(kept) && kept[i].Day.Before(s.Day) {
			i++
		}
		switch {
		case i < len(kept) && kept[i].Day == s.Day:
			kept[i] = s
		default:
			kept = append(kept[:i], append([]Snapshot{s}, kept[i:]...)...)
		}
	}
	return kept
}
