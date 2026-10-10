package application

import (
	"time"

	"visitron/internal/domain"
)

// DayOf is the calendar day of t in t's own zone, which the clock sets to the
// zone Windows is set to.
func DayOf(t time.Time) domain.Day {
	y, m, d := t.Date()
	return domain.Day{Year: y, Month: int(m), Date: d}
}

// DaysBack is the day n-1 days before t's day, so the n days ending on t's
// day start there.
func DaysBack(t time.Time, n int) domain.Day {
	return DayOf(t.AddDate(0, 0, -(n - 1)))
}

// DaysFrom lists every day from first up to t's day inclusive.
func DaysFrom(first domain.Day, t time.Time) []domain.Day {
	last := DayOf(t)
	day := time.Date(first.Year, time.Month(first.Month), first.Date, 0, 0, 0, 0, t.Location())
	var days []domain.Day
	for d := DayOf(day); !last.Before(d); d = DayOf(day) {
		days = append(days, d)
		day = day.AddDate(0, 0, 1)
	}
	return days
}
