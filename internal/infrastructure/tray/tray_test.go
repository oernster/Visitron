package tray

import "testing"

func TestCommandFor(t *testing.T) {
	t.Parallel()
	cases := map[uint32]Command{3: Show, 1: Refresh, 2: Quit}
	for id, want := range cases {
		if got, ok := commandFor(id, 3, 1, 2); !ok || got != want {
			t.Errorf("commandFor(%d) = %v %v; want %v", id, got, ok, want)
		}
	}
	for _, id := range []uint32{0, 99} {
		if _, ok := commandFor(id, 3, 1, 2); ok {
			t.Errorf("commandFor(%d) chose something", id)
		}
	}
}

func TestOfferNeverBlocks(t *testing.T) {
	t.Parallel()
	commands := make(chan Command, 1)
	offer(commands, Show)
	offer(commands, Quit) // the buffer is full: dropped, not blocked
	if got := <-commands; got != Show {
		t.Errorf("got %v", got)
	}
	if len(commands) != 0 {
		t.Error("a click beyond the buffer was kept")
	}
}
