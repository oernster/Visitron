// Command installer is the bespoke Visitron setup program.
//
// It is built as a second Wails application in the same module, so it wears the
// same WebView and the same palette as the application it installs. It carries
// the built application as an embedded zip and covers install, update, going
// back a version, repair, reinstall and uninstall, all per user with no
// administrator rights.
//
// Ported from ED Voyage Companion's installer, which is PigeonPost's model with
// the install policy moved into infrastructure.
package main

import (
	"embed"
	"os"
	"path/filepath"

	"github.com/wailsapp/wails/v2"
	"github.com/wailsapp/wails/v2/pkg/options"
	"github.com/wailsapp/wails/v2/pkg/options/assetserver"
	windowsoptions "github.com/wailsapp/wails/v2/pkg/options/windows"
	"visitron/internal/product"
)

//go:embed all:frontend/dist
var assets embed.FS

//go:embed payload.zip
var payload []byte

// appVersion is overridden at build time with -ldflags "-X main.appVersion=x.y.z",
// which only reaches a var. The setup program holds no version literal.
var appVersion = "dev"

const (
	windowTitle = product.Name + " Setup"
	// The window is fixed, so its height has to clear the tallest screen, which
	// is the licence one: a heading, what the licence means in five statements
	// and then the licence itself. Text that grows without the window growing
	// with it turns a fixed dialog into a scrolling one.
	//
	// 720 was measured against the install screen in a browser, where there is
	// no title bar. A real window spends about forty pixels of this on its
	// frame before the page sees any of it, which is why the licence screen
	// arrived with its last statement cut off and a scrollbar down the side.
	// The height is stated against the REAL client area now, with the frame
	// counted and slack left over.
	windowWidth  = 860
	windowHeight = 870
	// webviewFolder holds the setup window's own WebView2 cache, pinned under
	// TEMP rather than left to default into %APPDATA%, so running setup leaves
	// no folder beside the application's own.
	webviewFolder = product.Name + "Setup"
)

// background is the dark surface from the application's own palette. Setup
// opens dark, as the application does, so this is the ground it never flashes
// anything else over; a reader who moves it to light moves the page, which
// fills the window.
var background = options.RGBA{R: 0x14, G: 0x17, B: 0x1c, A: 1}

func main() {
	app := NewApp(payload, appVersion)
	_ = wails.Run(&options.App{
		Title:            windowTitle,
		Width:            windowWidth,
		Height:           windowHeight,
		DisableResize:    true,
		BackgroundColour: &background,
		AssetServer:      &assetserver.Options{Assets: assets},
		OnStartup:        app.startup,
		Bind:             []interface{}{app},
		Windows: &windowsoptions.Options{
			WebviewUserDataPath: filepath.Join(os.TempDir(), webviewFolder),
		},
	})
}
