//go:build windows

// Package windowfocus hands the keyboard to the page inside the window.
//
// Ported from PigeonPost's internal/infrastructure/taskbar/focus_windows.go,
// where this was measured and where it works. The lesson it carries: focusing
// the MAIN window is not enough, because WebView2 hosts the page in a child
// window of its own and the keys follow that child.
package windowfocus

import (
	"strings"
	"syscall"
	"unsafe"

	"golang.org/x/sys/windows"
)

// gwChild selects a window's first child; webviewHostClass is the WebView2 host
// window that must hold keyboard focus; the two maximums bound the name buffers.
const (
	gwChild           = 5 // GW_CHILD
	webviewHostClass  = "Chrome_WidgetWin_1"
	classNameMaxRune  = 256
	windowTextMaxRune = 512
)

var (
	moduser32               = windows.NewLazySystemDLL("user32.dll")
	procSetFocus            = moduser32.NewProc("SetFocus")
	procSetForegroundWindow = moduser32.NewProc("SetForegroundWindow")
	procAttachThreadInput   = moduser32.NewProc("AttachThreadInput")
	procEnumChildWindows    = moduser32.NewProc("EnumChildWindows")
	procGetClassName        = moduser32.NewProc("GetClassNameW")
	procGetWindow           = moduser32.NewProc("GetWindow")
	procGetWindowTextW      = moduser32.NewProc("GetWindowTextW")
)

// GiveTheKeyboardToThePage focuses this process's window and the WebView2
// control inside it, so the first key pressed reaches the page.
//
// Without it a cold launch drops every keystroke, the first Tab included, until
// somebody clicks in the window: WebView2 does not take keyboard focus when the
// window first appears. Focusing the main window does not fix it either, which
// is the part that is easy to get wrong.
//
// Every failure is ignored. The keyboard is worth a best effort and nothing
// more; a window that opens is better than one that refuses to.
func GiveTheKeyboardToThePage(title string) {
	hwnd := findMainWindow(title)
	if hwnd == 0 {
		return
	}
	target := findWebViewChild(hwnd)
	if target == 0 {
		target = hwnd
	}
	// SetFocus only takes effect from the thread owning the target window's
	// input queue, so this thread attaches to it for the duration. The window
	// comes to the foreground first; the focus change does not stick otherwise.
	winThread, _ := windows.GetWindowThreadProcessId(hwnd, nil)
	curThread := windows.GetCurrentThreadId()
	if winThread != 0 && winThread != curThread {
		_, _, _ = procAttachThreadInput.Call(uintptr(curThread), uintptr(winThread), 1)
		defer procAttachThreadInput.Call(uintptr(curThread), uintptr(winThread), 0)
	}
	_, _, _ = procSetForegroundWindow.Call(uintptr(hwnd))
	_, _, _ = procSetFocus.Call(uintptr(target))
}

// findMainWindow answers this process's visible window carrying title, else any
// visible window it owns.
func findMainWindow(title string) windows.HWND {
	pid := windows.GetCurrentProcessId()
	var exact, anyVisible windows.HWND
	cb := syscall.NewCallback(func(hwnd windows.HWND, _ uintptr) uintptr {
		var owner uint32
		_, _ = windows.GetWindowThreadProcessId(hwnd, &owner)
		if owner != pid || !windows.IsWindowVisible(hwnd) {
			return 1 // keep enumerating
		}
		if anyVisible == 0 {
			anyVisible = hwnd
		}
		if strings.EqualFold(windowText(hwnd), title) {
			exact = hwnd
			return 0 // stop
		}
		return 1
	})
	_ = windows.EnumWindows(cb, nil)
	if exact != 0 {
		return exact
	}
	return anyVisible
}

// findWebViewChild answers the WebView2 host nested in parent, else parent's
// first child. WebView2 hosts the page in a child window of its own class;
// focusing that child is what routes the keyboard into the page.
func findWebViewChild(parent windows.HWND) windows.HWND {
	var found windows.HWND
	cb := syscall.NewCallback(func(hwnd windows.HWND, _ uintptr) uintptr {
		if className(hwnd) == webviewHostClass {
			found = hwnd
			return 0 // stop enumerating
		}
		return 1
	})
	_, _, _ = procEnumChildWindows.Call(uintptr(parent), cb, 0)
	if found != 0 {
		return found
	}
	child, _, _ := procGetWindow.Call(uintptr(parent), gwChild)
	return windows.HWND(child)
}

// className answers a window's class name.
func className(hwnd windows.HWND) string {
	buf := make([]uint16, classNameMaxRune)
	n, _, _ := procGetClassName.Call(uintptr(hwnd), uintptr(unsafe.Pointer(&buf[0])), uintptr(len(buf)))
	return windows.UTF16ToString(buf[:n])
}

// windowText answers a window's title.
func windowText(hwnd windows.HWND) string {
	buf := make([]uint16, windowTextMaxRune)
	n, _, _ := procGetWindowTextW.Call(uintptr(hwnd), uintptr(unsafe.Pointer(&buf[0])), uintptr(len(buf)))
	return windows.UTF16ToString(buf[:n])
}
