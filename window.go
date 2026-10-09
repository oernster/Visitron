package main

import (
	"fmt"
	"os"
	"runtime/debug"
	"time"

	"github.com/oernster/visitron/internal/infrastructure/tray"
	"github.com/oernster/visitron/internal/infrastructure/windowfocus"
	"github.com/oernster/visitron/internal/product"
	"github.com/wailsapp/wails/v2/pkg/options"
	"github.com/wailsapp/wails/v2/pkg/runtime"
)

// settleBeforeFocus lets the window finish showing before the keyboard is
// handed over: PigeonPost's measured value, taken with the rest of this.
const settleBeforeFocus = 250 * time.Millisecond

// windowFocus hands the keyboard to the page, ported from SymDiary. Showing
// the main window is not enough: WebView2 hosts the page in a child window
// and the keys follow the child. The wait runs on its own goroutine with its
// own recover, so nothing about opening the window waits on it.
type windowFocus struct{}

// Focus gives the WebView2 control the keyboard shortly after opening.
func (windowFocus) Focus() {
	go func() {
		defer func() {
			if recovered := recover(); recovered != nil {
				fmt.Fprintf(os.Stderr, "handing over the keyboard: %v\n%s", recovered, debug.Stack())
			}
		}()
		time.Sleep(settleBeforeFocus)
		windowfocus.GiveTheKeyboardToThePage(product.Name)
	}()
}

// windowOpener hands an address to the desktop through the window's runtime.
type windowOpener struct{ app *App }

// Open opens address in the default browser.
func (o windowOpener) Open(address string) { runtime.BrowserOpenURL(o.app.ctx, address) }

// windowEmitter sends events to the page; before the window exists there is
// no page to tell, so nothing is sent.
type windowEmitter struct{ app *App }

// Emit sends one event.
func (e windowEmitter) Emit(name string, data any) {
	if e.app.ctx != nil {
		runtime.EventsEmit(e.app.ctx, name, data)
	}
}

// windowControls is the window's runtime as the facade's windowControl.
type windowControls struct{ app *App }

// Reveal brings the window back from the tray, the taskbar or behind others.
func (w windowControls) Reveal() {
	runtime.WindowUnminimise(w.app.ctx)
	runtime.Show(w.app.ctx)
}

// Hide takes the window off the screen and the taskbar; the tray stays.
func (w windowControls) Hide() { runtime.WindowHide(w.app.ctx) }

// Quit ends Visitron.
func (w windowControls) Quit() { runtime.Quit(w.app.ctx) }

// followTray acts on the tray's commands until it closes (FR-051). It runs on
// a goroutine of its own, so it carries its own recover.
func (a *App) followTray(commands <-chan tray.Command) {
	defer func() {
		if recovered := recover(); recovered != nil {
			fmt.Fprintf(os.Stderr, "panic following the tray: %v\n%s", recovered, debug.Stack())
		}
	}()
	for command := range commands {
		switch command {
		case tray.Show:
			a.control.Reveal()
		case tray.Refresh:
			if err := a.Refresh(); err != nil {
				fmt.Fprintf(os.Stderr, "refresh from the tray: %v\n", err)
			}
		case tray.Quit:
			a.quit()
		}
	}
}

// singleInstance brings the running window forward when Visitron is started
// again; the second copy then ends (FR-053).
func singleInstance(app *App) *options.SingleInstanceLock {
	return &options.SingleInstanceLock{
		UniqueId:               product.UniqueID,
		OnSecondInstanceLaunch: func(options.SecondInstanceData) { app.control.Reveal() },
	}
}
