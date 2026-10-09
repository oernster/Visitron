// Command visitron reports page loads on the owner's websites and downloads
// of their GitHub releases, with a daily history.
//
// This file is the composition root: the only file permitted to wire concrete
// infrastructure to the application layer.
package main

import (
	"embed"
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"time"

	"github.com/oernster/visitron/internal/application"
	"github.com/oernster/visitron/internal/infrastructure/github"
	"github.com/oernster/visitron/internal/infrastructure/goatcounter"
	"github.com/oernster/visitron/internal/infrastructure/runlog"
	"github.com/oernster/visitron/internal/infrastructure/secrets"
	"github.com/oernster/visitron/internal/infrastructure/startup"
	"github.com/oernster/visitron/internal/infrastructure/store"
	"github.com/oernster/visitron/internal/infrastructure/web"
	"github.com/oernster/visitron/internal/product"
	"github.com/wailsapp/wails/v2"
	"github.com/wailsapp/wails/v2/pkg/options"
	"github.com/wailsapp/wails/v2/pkg/options/assetserver"
)

//go:embed all:frontend/dist
var assets embed.FS

// seedFile is the record of every site carrying the counter (FR-011).
//
//go:embed sites.seed.json
var seedFile []byte

// appVersion is set from VERSION by build.ps1 through -ldflags -X, which only
// reaches a var.
var appVersion = "0.0.0-dev"

// Window geometry: wide enough for the list beside the detail.
const (
	windowWidth     = 1200
	windowHeight    = 800
	windowMinWidth  = 860
	windowMinHeight = 600
)

// problemLayout words data that could not be opened.
const problemLayout = "Visitron's data could not be opened, so nothing can be shown " +
	"until this is put right. Nothing in the file has been changed. %v"

// systemClock is the real clock in the zone Windows is set to.
type systemClock struct{}

// Now answers the current instant.
func (systemClock) Now() time.Time { return time.Now() }

func main() {
	keepLog(time.Now())

	var data application.Store
	closeData := func() error { return nil }
	problem := ""
	path, err := store.DefaultPath()
	if err == nil {
		var opened *store.Store
		if opened, err = store.Open(path); err == nil {
			data, closeData = opened, opened.Close
		}
	}
	if err != nil {
		// The window opens regardless and says what went wrong (house rule 1).
		fmt.Fprintf(os.Stderr, "opening the data: %v\n", err)
		problem = fmt.Sprintf(problemLayout, err)
		data = store.Unavailable{Reason: err}
	}
	seed, err := application.ParseSeed(seedFile)
	if err != nil {
		fmt.Fprintf(os.Stderr, "the embedded seed: %v\n", err)
	}

	clock := systemClock{}
	client := web.NewClient()
	vault := secrets.Vault{}
	releases := github.NewClient(client, vault, github.DefaultBase)
	loads := goatcounter.NewClient(client, product.GoatCounterSite)
	exe, _ := os.Executable()
	entry := startup.Entry{Name: product.Name, Exe: filepath.Clean(exe)}
	check := application.NewCheck(data, releases, loads, vault, clock)
	services := Services{
		Websites:  application.NewWebsites(data, web.NewFetcher(client), releases),
		Figures:   application.NewFigures(data, clock),
		Scheduler: application.NewScheduler(check, data, clock),
		Settings:  application.NewSettings(data, vault, entry, releases, loads),
		Store:     data,
		Seed:      seed,
	}
	app := newApp(services, appVersion, problem, closeData)
	app.focuser = windowFocus{}
	app.opener = windowOpener{app: app}
	app.emitter = windowEmitter{app: app}

	err = wails.Run(&options.App{
		Title:       product.Name,
		Width:       windowWidth,
		Height:      windowHeight,
		MinWidth:    windowMinWidth,
		MinHeight:   windowMinHeight,
		StartHidden: slices.Contains(os.Args[1:], startup.HiddenFlag),
		// FR-050 (close to tray) waits for the tray icon: until it exists, a
		// hidden window could not be reopened or quit, so closing quits.
		HideWindowOnClose:  false,
		AssetServer:        &assetserver.Options{Assets: assets},
		OnStartup:          app.startup,
		OnDomReady:         app.ready,
		OnShutdown:         app.shutdown,
		SingleInstanceLock: singleInstance(app),
		Bind:               []interface{}{app},
	})
	if err != nil {
		fmt.Fprintf(os.Stderr, "running the window: %v\n", err)
		os.Exit(1)
	}
}

// keepLog opens the run log and points the error output at it before anything
// else can fail (NFR-REL-002). A log that cannot be kept does not stop the run.
func keepLog(started time.Time) {
	path, err := runlog.Path()
	if err != nil {
		return
	}
	log, err := runlog.Open(path, started)
	if err != nil {
		return
	}
	if err := runlog.Keep(log); err != nil {
		fmt.Fprintf(log, "keeping the log: %v\n", err)
	}
}
