//go:build !windows

// Package windowfocus hands the keyboard to the page inside the window.
package windowfocus

// GiveTheKeyboardToThePage does nothing off Windows.
//
// The defect it answers is Windows-only: WebView2 hosts the page in a child
// window that does not take keyboard focus when the window first appears. The
// webkit2gtk and WebKit backends Wails uses elsewhere hand the page the
// keyboard themselves, so there is nothing to put right here.
func GiveTheKeyboardToThePage(string) {}
