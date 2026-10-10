package domain

import (
	"errors"
	"fmt"
)

// Check interval bounds in whole hours (FR-063).
const (
	hoursPerDay          = 24
	MinIntervalHours     = 1
	MaxIntervalHours     = int(Week) * hoursPerDay
	DefaultIntervalHours = hoursPerDay
)

// ErrInterval refuses a check interval outside its bounds.
var ErrInterval = fmt.Errorf("the check interval is whole hours from %d to %d",
	MinIntervalHours, MaxIntervalHours)

// ValidInterval accepts a check interval in hours, else refuses it.
func ValidInterval(hours int) error {
	if hours < MinIntervalHours || hours > MaxIntervalHours {
		return ErrInterval
	}
	return nil
}

// The bounds of the owner's own downloads of each macOS disk image, taken off
// its count (Amendment 19). None is the default, since most owners download
// nothing of their own; the top is a guard against a slip of the finger
// emptying every disk image's count, not a measured limit.
const (
	MinSelfDownloads     = 0
	MaxSelfDownloads     = 10
	DefaultSelfDownloads = MinSelfDownloads
)

// ErrSelfDownloads refuses an own-download count outside its bounds.
var ErrSelfDownloads = fmt.Errorf("your own downloads of each disk image are a whole number from %d to %d",
	MinSelfDownloads, MaxSelfDownloads)

// ValidSelfDownloads accepts an own-download count, else refuses it.
func ValidSelfDownloads(n int) error {
	if n < MinSelfDownloads || n > MaxSelfDownloads {
		return ErrSelfDownloads
	}
	return nil
}

// Period is a span of days the window reports over (FR-043).
type Period int

// The periods offered, in days.
const (
	Week    Period = 7
	Month   Period = 30
	Quarter Period = 90
	Year    Period = 365
)

// Periods lists the periods in the order the window offers them.
var Periods = []Period{Week, Month, Quarter, Year}

// DefaultPeriod is the period shown until the owner picks another.
const DefaultPeriod = Month

// ErrPeriod refuses a period that is not offered.
var ErrPeriod = errors.New("that period is not offered")

// ValidPeriod accepts an offered period, else refuses it.
func ValidPeriod(days int) (Period, error) {
	for _, p := range Periods {
		if int(p) == days {
			return p, nil
		}
	}
	return 0, ErrPeriod
}

// Theme is the window's colour mode (FR-070).
type Theme string

// The two themes; dark is the default.
const (
	Dark         Theme = "dark"
	Light        Theme = "light"
	DefaultTheme       = Dark
)

// Next is the theme the theme button switches to.
func (t Theme) Next() Theme {
	if t == Light {
		return Dark
	}
	return Light
}
