package main

import "context"

// Closing the window asks rather than assumes, ported from PigeonPost: the
// close button cancels the close and the page offers Minimise to tray or Quit
// (FR-050, Amendment 4).

// closeRequestEvent asks the page to offer the close choice.
const closeRequestEvent = "close-request"

// windowControl shows, hides and ends the window. The window's runtime
// implements it in window.go; a test supplies its own.
type windowControl interface {
	Reveal()
	Hide()
	Quit()
}

// beforeClose runs when the window's close button is pressed. It answers true
// to keep the window, having asked the page for the choice; false lets the
// close go ahead, which it does when a Quit is already under way or when there
// is no tray icon to come back to.
func (a *App) beforeClose(context.Context) bool {
	if a.quitting.Load() || !a.trayUp {
		return false
	}
	// A close asked for from the taskbar while the window is minimised or
	// behind others would otherwise ask on a window nobody can see.
	a.control.Reveal()
	a.emit(closeRequestEvent, nil)
	return true
}

// MinimiseToTray hides the window; Visitron keeps checking from the tray.
func (a *App) MinimiseToTray() (err error) {
	defer guard(&err)
	a.control.Hide()
	return nil
}

// RequestQuit ends Visitron from the close choice.
func (a *App) RequestQuit() (err error) {
	defer guard(&err)
	a.quit()
	return nil
}

// quit records that a Quit is under way, so closing asks nothing, then ends.
func (a *App) quit() {
	a.quitting.Store(true)
	a.control.Quit()
}
