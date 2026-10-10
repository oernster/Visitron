package application

import (
	"testing"

	"visitron/internal/domain"
)

func TestDaysAcrossTheClockChange(t *testing.T) {
	t.Parallel()
	// The clocks went back in London on 2026-10-25; the day still counts once.
	now := at(2026, 10, 26, 0)
	days := DaysFrom(domain.Day{Year: 2026, Month: 10, Date: 24}, now)
	if len(days) != 3 || days[1] != (domain.Day{Year: 2026, Month: 10, Date: 25}) {
		t.Errorf("days %v", days)
	}
	if got := DaysBack(now, 3); got != (domain.Day{Year: 2026, Month: 10, Date: 24}) {
		t.Errorf("DaysBack = %v", got)
	}
	if got := DaysFrom(domain.Day{Year: 2026, Month: 10, Date: 27}, now); len(got) != 0 {
		t.Errorf("a first day after today gave %v", got)
	}
}
