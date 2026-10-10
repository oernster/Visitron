package domain

import "testing"

func TestReleaseVersion(t *testing.T) {
	t.Parallel()
	for tag, want := range map[string]string{"v1.2.0": "1.2.0", " 1.2.0 ": "1.2.0", "1.2.0": "1.2.0", "V1.2.0": "V1.2.0"} {
		if got := ReleaseVersion(tag); got != want {
			t.Errorf("ReleaseVersion(%q) = %q, want %q", tag, got, want)
		}
	}
}

func TestNewer(t *testing.T) {
	t.Parallel()
	cases := []struct {
		latest, running   string
		newer, comparable bool
	}{
		{"1.2.0", "1.1.9", true, true},
		{"1.2.0", "1.2.0", false, true},
		{"1.1.0", "1.2.0", false, true},
		{"1.10.0", "1.9.0", true, true},
		{"1.5.0.1", "1.5.0", true, true},
		{"1.5.0", "1.5", false, true},
		{"1.6", "1.5.9", true, true},
		{"1.6.0-beta", "1.5.0", false, true},
		{"", "1.5.0", false, true},
		{"1.6.0", "0.0.0-dev", false, false},
		{"1.6.0", "", false, false},
		{"-1.6.0", "1.5.0", false, true},
	}
	for _, c := range cases {
		newer, comparable := Newer(c.latest, c.running)
		if newer != c.newer || comparable != c.comparable {
			t.Errorf("Newer(%q, %q) = %v, %v; want %v, %v", c.latest, c.running, newer, comparable, c.newer, c.comparable)
		}
	}
}
