package main

import (
	"context"
	"errors"
	"fmt"
	"os"
	"runtime/debug"
	"sync"
	"sync/atomic"
	"time"

	"github.com/oernster/visitron/internal/application"
	"github.com/oernster/visitron/internal/product"
)

// errInternal is what the page is told when a bound method panics. The stack
// goes to the log; the page gets a sentence it can show.
//
//lint:ignore ST1005 the page shows this sentence as it stands; it opens with the product's name
var errInternal = errors.New(product.Name + " hit an internal fault; the details are in its log")

// tickEvery is how often the scheduler asks whether a check is due. It is
// also what makes an overdue check start within a minute of a resume from
// sleep (FR-031): the next tick after the clock jumps finds it due.
const tickEvery = time.Minute

// progressEvent is the event the page listens to while a check runs.
const progressEvent = "check-progress"

// windowFocuser hands the window's keyboard to the page, ported from SymDiary:
// WebView2 holds DOM focus and keyboard focus apart.
type windowFocuser interface{ Focus() }

// browserOpener hands an address to the desktop's browser.
type browserOpener interface{ Open(address string) }

// emitter sends an event to the page.
type emitter interface{ Emit(name string, data any) }

// Services is what the facade drives: one application service per concern.
type Services struct {
	Websites  *application.Websites
	Figures   *application.Figures
	Scheduler *application.Scheduler
	Settings  *application.Settings
	Updates   *application.Updates
	Store     application.Store
	Seed      []application.Website
}

// App is the facade the window calls. It converts between the page's shapes
// and the application's; every rule lives below it.
type App struct {
	ctx      context.Context
	cancel   context.CancelFunc
	version  string
	problem  string
	services Services
	focuser  windowFocuser
	opener   browserOpener
	emitter  emitter
	control  windowControl
	close    func() error
	// trayUp is whether the tray icon came up: without it, hiding the window
	// would leave nothing to bring it back, so closing quits (FR-050).
	trayUp bool
	// quitting is set once a Quit is under way, so closing asks nothing.
	quitting atomic.Bool
	// offered is the release the last update check offered; Download and Skip
	// act on it, so the page never holds an address (Amendment 5).
	mu      sync.Mutex
	offered application.UpdateStatus
	// followers each run on a goroutine of their own once the window has
	// started, such as following the tray icon's clicks. The composition root
	// supplies them, so the facade imports no infrastructure.
	followers []func()
}

// newApp answers the facade. problem is the reason the data could not be
// opened, empty when it opened; the page shows it.
func newApp(services Services, version, problem string, close func() error) *App {
	return &App{version: version, problem: problem, services: services, close: close}
}

// startup keeps the window's context, seeds a first run (FR-011) and starts
// the scheduler.
func (a *App) startup(ctx context.Context) {
	a.ctx, a.cancel = context.WithCancel(ctx)
	if _, err := a.services.Websites.Seed(a.services.Seed); err != nil {
		fmt.Fprintf(os.Stderr, "seeding the websites: %v\n", err)
	}
	go a.schedule()
	for _, follow := range a.followers {
		go follow()
	}
}

// schedule asks every tickEvery whether a check is due, at once included.
// It runs on a goroutine of its own, so it carries its own recover; a fault is
// logged and the next tick tries again rather than the loop ending silently.
func (a *App) schedule() {
	ticker := time.NewTicker(tickEvery)
	defer ticker.Stop()
	for {
		a.tickOnce()
		select {
		case <-a.ctx.Done():
			return
		case <-ticker.C:
		}
	}
}

// tickOnce runs one tick. The closing event is deferred so the page stops
// saying a check is under way however the tick ended.
func (a *App) tickOnce() {
	defer a.emit(progressEvent, ProgressDTO{})
	defer a.survive("the scheduler")
	if _, _, err := a.services.Scheduler.Tick(a.ctx, a.progress); err != nil {
		fmt.Fprintf(os.Stderr, "scheduled check: %v\n", err)
	}
}

// survive is deferred at the top of a goroutine that runs a check. A panic is
// logged with its stack, then recorded as a failed check so the window warns
// of it (house robustness rule 8, Amendment 3).
func (a *App) survive(where string) {
	recovered := recover()
	if recovered == nil {
		return
	}
	fmt.Fprintf(os.Stderr, "panic in %s: %v\n%s", where, recovered, debug.Stack())
	if a.services.Scheduler == nil {
		return
	}
	if err := a.services.Scheduler.RecordFault(errInternal.Error()); err != nil {
		fmt.Fprintf(os.Stderr, "recording the fault: %v\n", err)
	}
}

// progress tells the page how far a check has got (FR-033).
func (a *App) progress(done, total int, site string) {
	a.emit(progressEvent, ProgressDTO{Done: done, Total: total, Site: site})
}

func (a *App) emit(name string, data any) {
	if a.emitter != nil {
		a.emitter.Emit(name, data)
	}
}

// ready hands the keyboard to the page once it exists (keeb: the host must).
func (a *App) ready(context.Context) {
	if a.focuser == nil {
		fmt.Fprintln(os.Stderr, "the window has no focuser, so the page starts without the keyboard")
		return
	}
	a.focuser.Focus()
}

// shutdown stops the scheduler and closes the data.
func (a *App) shutdown(context.Context) {
	if a.cancel != nil {
		a.cancel()
	}
	if err := a.close(); err != nil {
		fmt.Fprintf(os.Stderr, "closing the data: %v\n", err)
	}
}

// guard turns a panic in a bound method into errInternal, writing the stack to
// the log. A bound method runs on a goroutine Wails owns, so the recover sits
// in the method itself.
func guard(err *error) {
	if recovered := recover(); recovered != nil {
		fmt.Fprintf(os.Stderr, "panic in a bound method: %v\n%s", recovered, debug.Stack())
		*err = errInternal
	}
}
