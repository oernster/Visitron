//go:build windows

package tray

import (
	"fmt"
	"os"
	"runtime"
	"runtime/debug"
	"sync"
	"sync/atomic"
	"unsafe"

	"golang.org/x/sys/windows"

	"visitron/internal/product"
)

// className is the hidden window the icon talks through.
const className = product.Name + "TrayWindow"

// Tray is the icon, its hidden window and the thread that pumps its messages.
type Tray struct {
	commands chan Command
	window   windows.HWND
	posted   atomic.Uintptr
	icon     windows.Handle
	started  sync.Once
	stopped  sync.Once
	ready    chan error
	// taskbarCreated is the shell's TaskbarCreated message, on which the icon
	// is added again; zero where it could not be registered.
	taskbarCreated uint32
}

// New builds a tray; Start shows it.
func New() *Tray {
	return &Tray{commands: make(chan Command, commandBuffer), ready: make(chan error, 1)}
}

// Commands delivers the owner's choices; it closes when the tray stops.
func (t *Tray) Commands() <-chan Command { return t.commands }

// Start shows the icon on a locked OS thread of its own and answers once it
// is up or has failed.
func (t *Tray) Start() error {
	var err error
	t.started.Do(func() {
		go t.run()
		err = <-t.ready
	})
	return err
}

// Stop removes the icon and ends its thread.
func (t *Tray) Stop() {
	t.stopped.Do(func() {
		if window := t.posted.Load(); window != 0 {
			_, _, _ = procPostMessage.Call(window, wmClose, 0, 0)
		}
	})
}

func executablePath() string {
	path, err := os.Executable()
	if err != nil {
		return ""
	}
	return path
}

// run owns the window: a Win32 window answers only on the thread that made it.
func (t *Tray) run() {
	runtime.LockOSThread()
	defer runtime.UnlockOSThread()
	defer close(t.commands)
	defer func() {
		if recovered := recover(); recovered != nil {
			fmt.Fprintf(os.Stderr, "panic in the tray: %v\n%s", recovered, debug.Stack())
		}
	}()
	if err := t.create(); err != nil {
		t.ready <- err
		return
	}
	t.ready <- nil
	t.pump()
	t.destroy()
}

func (t *Tray) create() error {
	t.registerTaskbarCreated()
	instance, _, _ := procGetModuleHandle.Call(0)
	name, err := windows.UTF16PtrFromString(className)
	if err != nil {
		return fmt.Errorf("encoding the tray class name: %w", err)
	}
	class := wndClassEx{lpfnWndProc: windows.NewCallback(t.windowProc), hInstance: windows.Handle(instance), lpszClassName: name}
	class.cbSize = uint32(unsafe.Sizeof(class))
	if ret, _, callErr := procRegisterClassEx.Call(uintptr(unsafe.Pointer(&class))); ret == 0 {
		return fmt.Errorf("registering the tray window class: %w", callErr)
	}
	handle, _, callErr := procCreateWindowEx.Call(0, uintptr(unsafe.Pointer(name)), uintptr(unsafe.Pointer(name)),
		0, 0, 0, 0, 0, 0, 0, instance, 0)
	if handle == 0 {
		return fmt.Errorf("creating the tray window: %w", callErr)
	}
	t.window = windows.HWND(handle)
	t.icon = ownIcon()
	if err := t.addIcon(); err != nil {
		_, _, _ = procDestroyWindow.Call(handle)
		t.window = 0
		return err
	}
	t.posted.Store(handle)
	return nil
}

// registerTaskbarCreated learns the shell's TaskbarCreated message, so the
// icon comes back when Explorer restarts rather than leaving a process the
// owner cannot reach. Without it the icon still shows; it is only not restored.
func (t *Tray) registerTaskbarCreated() {
	name, err := windows.UTF16PtrFromString(taskbarCreatedMessage)
	if err != nil {
		return
	}
	message, _, callErr := procRegisterWindowMsg.Call(uintptr(unsafe.Pointer(name)))
	if message == 0 {
		fmt.Fprintf(os.Stderr, "registering %s: %v\n", taskbarCreatedMessage, callErr)
		return
	}
	t.taskbarCreated = uint32(message)
}

// addIcon puts the icon in the notification area.
func (t *Tray) addIcon() error {
	data := t.iconData()
	if ret, _, callErr := procShellNotifyIcon.Call(nimAdd, uintptr(unsafe.Pointer(&data))); ret == 0 {
		return fmt.Errorf("adding the tray icon: %w", callErr)
	}
	return nil
}

func (t *Tray) iconData() notifyIconData {
	data := notifyIconData{hWnd: t.window, uID: trayIconID, uFlags: nifMessage | nifIcon | nifTip,
		uCallbackMessage: wmTrayCallback, hIcon: t.icon}
	data.cbSize = uint32(unsafe.Sizeof(data))
	copyUTF16(data.szTip[:], product.Name)
	return data
}

func (t *Tray) pump() {
	var message msg
	for {
		ret, _, _ := procGetMessage.Call(uintptr(unsafe.Pointer(&message)), 0, 0, 0)
		if int32(ret) <= 0 {
			return
		}
		_, _, _ = procTranslateMessage.Call(uintptr(unsafe.Pointer(&message)))
		_, _, _ = procDispatchMessage.Call(uintptr(unsafe.Pointer(&message)))
	}
}

func (t *Tray) destroy() {
	if t.window == 0 {
		return
	}
	t.posted.Store(0)
	data := t.iconData()
	_, _, _ = procShellNotifyIcon.Call(nimDelete, uintptr(unsafe.Pointer(&data)))
	_, _, _ = procDestroyWindow.Call(uintptr(t.window))
	t.window = 0
}

func (t *Tray) windowProc(hwnd windows.HWND, message uint32, wParam, lParam uintptr) uintptr {
	if t.taskbarCreated != 0 && message == t.taskbarCreated {
		if err := t.addIcon(); err != nil {
			fmt.Fprintf(os.Stderr, "restoring the tray icon after the taskbar restarted: %v\n", err)
		}
		return 0
	}
	switch message {
	case wmTrayCallback:
		switch lParam {
		case wmRButtonUp:
			t.showMenu()
		case wmLButtonUp, wmLButtonDblClk:
			offer(t.commands, Show)
		}
		return 0
	case wmClose:
		_, _, _ = procDestroyWindow.Call(uintptr(hwnd))
		return 0
	case wmDestroy:
		_, _, _ = procPostQuitMessage.Call(0)
		return 0
	}
	ret, _, _ := procDefWindowProc.Call(uintptr(hwnd), uintptr(message), wParam, lParam)
	return ret
}

// showMenu pops the menu at the pointer, as Bridge Talk's does: the window
// goes to the foreground first, else the menu will not close when the owner
// clicks elsewhere.
func (t *Tray) showMenu() {
	menu, _, _ := procCreatePopupMenu.Call()
	if menu == 0 {
		return
	}
	defer func() { _, _, _ = procDestroyMenu.Call(menu) }()
	appendMenuItem(menu, idShow, openItem, 0)
	appendMenuItem(menu, idRefresh, refreshItem, 0)
	appendSeparator(menu)
	appendMenuItem(menu, idQuit, quitItem, 0)
	var cursor point
	_, _, _ = procGetCursorPos.Call(uintptr(unsafe.Pointer(&cursor)))
	_, _, _ = procSetForegroundWindow.Call(uintptr(t.window))
	chosen, _, _ := procTrackPopupMenu.Call(menu, tpmRightButton|tpmNonotify|tpmReturnCmd,
		uintptr(cursor.x), uintptr(cursor.y), 0, uintptr(t.window), 0)
	_, _, _ = procPostMessage.Call(uintptr(t.window), wmNull, 0, 0)
	if command, ok := commandFor(uint32(chosen), idShow, idRefresh, idQuit); ok {
		offer(t.commands, command)
	}
}
