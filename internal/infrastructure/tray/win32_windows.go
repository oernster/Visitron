//go:build windows

package tray

import (
	"unsafe"

	"golang.org/x/sys/windows"
)

// Win32 window-message, shell-notify and menu constants used by the tray.
const (
	wmApp           = 0x8000
	wmTrayCallback  = wmApp + 1 // the tray icon posts its mouse events here
	wmNull          = 0x0000
	wmDestroy       = 0x0002
	wmClose         = 0x0010
	wmLButtonUp     = 0x0202
	wmLButtonDblClk = 0x0203
	wmRButtonUp     = 0x0205

	nimAdd    = 0x0
	nimModify = 0x1
	nimDelete = 0x2

	nifMessage = 0x01
	nifIcon    = 0x02
	nifTip     = 0x04

	mfString    = 0x0000
	mfSeparator = 0x0800

	tpmRightButton = 0x0002
	tpmNonotify    = 0x0080
	tpmReturnCmd   = 0x0100

	idiApplication = 32512
	trayIconID     = 1

	// Menu command identifiers (FR-051).
	idRefresh = 1
	idQuit    = 2
	idShow    = 3
)

var (
	moduser32   = windows.NewLazySystemDLL("user32.dll")
	modshell32  = windows.NewLazySystemDLL("shell32.dll")
	modkernel32 = windows.NewLazySystemDLL("kernel32.dll")

	procShellNotifyIcon     = modshell32.NewProc("Shell_NotifyIconW")
	procExtractIconEx       = modshell32.NewProc("ExtractIconExW")
	procGetModuleHandle     = modkernel32.NewProc("GetModuleHandleW")
	procRegisterClassEx     = moduser32.NewProc("RegisterClassExW")
	procCreateWindowEx      = moduser32.NewProc("CreateWindowExW")
	procDestroyWindow       = moduser32.NewProc("DestroyWindow")
	procDefWindowProc       = moduser32.NewProc("DefWindowProcW")
	procGetMessage          = moduser32.NewProc("GetMessageW")
	procTranslateMessage    = moduser32.NewProc("TranslateMessage")
	procDispatchMessage     = moduser32.NewProc("DispatchMessageW")
	procPostQuitMessage     = moduser32.NewProc("PostQuitMessage")
	procPostMessage         = moduser32.NewProc("PostMessageW")
	procCreatePopupMenu     = moduser32.NewProc("CreatePopupMenu")
	procAppendMenu          = moduser32.NewProc("AppendMenuW")
	procTrackPopupMenu      = moduser32.NewProc("TrackPopupMenu")
	procDestroyMenu         = moduser32.NewProc("DestroyMenu")
	procGetCursorPos        = moduser32.NewProc("GetCursorPos")
	procLoadIcon            = moduser32.NewProc("LoadIconW")
	procSetForegroundWindow = moduser32.NewProc("SetForegroundWindow")
)

// notifyIconData mirrors NOTIFYICONDATAW. cbSize is set to the whole structure so
// the shell treats it as the modern version.
type notifyIconData struct {
	cbSize            uint32
	hWnd              windows.HWND
	uID               uint32
	uFlags            uint32
	uCallbackMessage  uint32
	hIcon             windows.Handle
	szTip             [128]uint16
	dwState           uint32
	dwStateMask       uint32
	szInfo            [256]uint16
	uVersionOrTimeout uint32
	szInfoTitle       [64]uint16
	dwInfoFlags       uint32
	guidItem          windows.GUID
	hBalloonIcon      windows.Handle
}

// wndClassEx mirrors WNDCLASSEXW for registering the hidden owner-window class.
type wndClassEx struct {
	cbSize        uint32
	style         uint32
	lpfnWndProc   uintptr
	cbClsExtra    int32
	cbWndExtra    int32
	hInstance     windows.Handle
	hIcon         windows.Handle
	hCursor       windows.Handle
	hbrBackground windows.Handle
	lpszMenuName  *uint16
	lpszClassName *uint16
	hIconSm       windows.Handle
}

// point mirrors POINT.
type point struct{ x, y int32 }

// msg mirrors MSG, pumped by the tray thread's message loop.
type msg struct {
	hwnd     windows.HWND
	message  uint32
	wParam   uintptr
	lParam   uintptr
	time     uint32
	pt       point
	lPrivate uint32
}

// copyUTF16 writes text into a fixed-size UTF-16 field, always leaving it
// null-terminated even when the text is too long and has to be truncated.
func copyUTF16(dst []uint16, text string) {
	encoded, err := windows.UTF16FromString(text)
	if err != nil {
		return
	}
	if copy(dst, encoded) >= len(dst) {
		dst[len(dst)-1] = 0
	}
}

// appendMenuItem adds a text command item to a popup menu.
func appendMenuItem(menu uintptr, id uint32, label string, flags uintptr) {
	pointer, err := windows.UTF16PtrFromString(label)
	if err != nil {
		return
	}
	_, _, _ = procAppendMenu.Call(menu, mfString|flags, uintptr(id), uintptr(unsafe.Pointer(pointer)))
}

// appendSeparator adds a divider to a popup menu.
func appendSeparator(menu uintptr) {
	_, _, _ = procAppendMenu.Call(menu, mfSeparator, 0, 0)
}

// ownIcon loads the icon embedded in this executable, falling back to the shell's
// generic application icon when the binary carries none. A build without an icon
// resource is the normal case before packaging, so this must not fail.
func ownIcon() windows.Handle {
	executable, err := windows.UTF16PtrFromString(executablePath())
	if err == nil {
		var large, small windows.Handle
		ret, _, _ := procExtractIconEx.Call(
			uintptr(unsafe.Pointer(executable)),
			0,
			uintptr(unsafe.Pointer(&large)),
			uintptr(unsafe.Pointer(&small)),
			1,
		)
		if ret > 0 && small != 0 {
			return small
		}
		if ret > 0 && large != 0 {
			return large
		}
	}
	handle, _, _ := procLoadIcon.Call(0, uintptr(idiApplication))
	return windows.Handle(handle)
}
