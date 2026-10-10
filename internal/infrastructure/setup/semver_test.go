package setup

import "testing"

func TestCompareDecidesTheRoute(t *testing.T) {
	t.Parallel()
	cases := []struct {
		carried, installed string
		want               Relation
	}{
		{"1.0.0", "1.0.0", Same},
		{"1.2.0", "1.1.9", Newer},
		{"1.1.9", "1.2.0", Older},
		{"2.0.0", "1.9.9", Newer},
		{"0.1.0", "0.1.1", Older},
		{"1.0.0-beta", "1.0.0", Same},
		{"1.0", "1.0.0", Same},
		{"", "", Same},
		{"rubbish", "0.0.0", Same},
		{"1.0.0", "rubbish", Newer},
	}
	for _, c := range cases {
		if got := Compare(c.carried, c.installed); got != c.want {
			t.Errorf("Compare(%q, %q) = %d, want %d", c.carried, c.installed, got, c.want)
		}
	}
}
