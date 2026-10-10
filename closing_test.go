package main

import (
	"context"
	"slices"
	"testing"

	"visitron/internal/infrastructure/tray"
)

func TestCloseAsksWhenTheTrayIsUp(t *testing.T) {
	r := newRig(t, nil)
	r.app.trayUp = true
	if !r.app.beforeClose(context.Background()) {
		t.Fatal("the close went ahead without asking")
	}
	if r.window.revealed != 1 || !slices.Contains(r.window.names, closeRequestEvent) {
		t.Fatalf("revealed %d, events %v: the page was not asked in view", r.window.revealed, r.window.names)
	}
}

func TestCloseQuitsWithoutATray(t *testing.T) {
	r := newRig(t, nil)
	if r.app.beforeClose(context.Background()) {
		t.Fatal("with no tray icon, closing kept a window nobody could bring back")
	}
	if len(r.window.names) != 0 {
		t.Fatalf("the page was asked anyway: %v", r.window.names)
	}
}

func TestMinimiseHidesAndQuitDoesNotAskAgain(t *testing.T) {
	r := newRig(t, nil)
	r.app.trayUp = true
	if err := r.app.MinimiseToTray(); err != nil || r.window.hidden != 1 {
		t.Fatalf("MinimiseToTray: %v, hidden %d", err, r.window.hidden)
	}
	if err := r.app.RequestQuit(); err != nil || r.window.quits != 1 {
		t.Fatalf("RequestQuit: %v, quits %d", err, r.window.quits)
	}
	if r.app.beforeClose(context.Background()) {
		t.Fatal("the close that a Quit causes asked again")
	}
}

func TestTheTrayRevealsAndQuitsThroughTheSameWindow(t *testing.T) {
	r := newRig(t, nil)
	r.app.trayUp = true
	commands := make(chan tray.Command, 3)
	commands <- tray.Show
	commands <- tray.Refresh
	commands <- tray.Quit
	close(commands)
	r.app.followTray(commands)
	if r.window.revealed != 1 || r.window.quits != 1 {
		t.Fatalf("revealed %d, quits %d", r.window.revealed, r.window.quits)
	}
	if r.app.beforeClose(context.Background()) {
		t.Fatal("Quit from the tray left the close asking")
	}
}
