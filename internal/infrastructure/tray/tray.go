// Package tray is Visitron's notification-area icon (FR-050, FR-051), ported
// from Bridge Talk's Win32 tray and trimmed to Open, Refresh now and Quit. A
// left click opens the window; a right click shows the menu.
package tray

// Command is what the owner chose from the tray.
type Command int

// The tray's commands.
const (
	Show Command = iota
	Refresh
	Quit
)

// The menu's words.
const (
	openItem    = "Open"
	refreshItem = "Refresh now"
	quitItem    = "Quit"
)

// commandBuffer lets a few clicks wait while the window is busy rather than
// block the tray's own thread.
const commandBuffer = 8

// commandFor maps a menu choice to a command; ok is false for a dismissed
// menu or an id the menu never offered.
func commandFor(chosen uint32, show, refresh, quit uint32) (Command, bool) {
	switch chosen {
	case show:
		return Show, true
	case refresh:
		return Refresh, true
	case quit:
		return Quit, true
	}
	return 0, false
}

// offer sends a command without ever blocking the tray's thread; a full
// buffer drops the click rather than freezing the icon.
func offer(commands chan<- Command, command Command) {
	select {
	case commands <- command:
	default:
	}
}
