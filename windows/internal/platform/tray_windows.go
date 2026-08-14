//go:build windows

package platform

import (
	"fmt"
	"unsafe"

	"github.com/zzl/go-win32api/v2/win32"
)

type trayIcon struct {
	app   *App
	data  win32.NOTIFYICONDATAW
	added bool
}

func newTrayIcon(app *App) *trayIcon {
	tray := &trayIcon{app: app}
	tray.data.CbSize = uint32(unsafe.Sizeof(tray.data))
	tray.data.HWnd = app.controllerWindow
	tray.data.UID = 1
	tray.data.UFlags = win32.NIF_MESSAGE | win32.NIF_ICON | win32.NIF_TIP
	tray.data.UCallbackMessage = wmTray
	tray.data.HIcon = app.icon
	copyUTF16(tray.data.SzTip[:], "Region Lockpaw")
	return tray
}

func (tray *trayIcon) add() error {
	if tray.added {
		win32.Shell_NotifyIconW(win32.NIM_DELETE, &tray.data)
		tray.added = false
	}
	if win32.Shell_NotifyIconW(win32.NIM_ADD, &tray.data) == win32.FALSE {
		return fmt.Errorf("add notification-area icon")
	}
	tray.added = true
	return nil
}

func (tray *trayIcon) remove() {
	if !tray.added {
		return
	}
	win32.Shell_NotifyIconW(win32.NIM_DELETE, &tray.data)
	tray.added = false
}

func (tray *trayIcon) showBalloon(title, message string, warning bool) {
	if !tray.added {
		return
	}
	tray.data.UFlags = win32.NIF_INFO
	tray.data.DwInfoFlags = win32.NIIF_INFO
	if warning {
		tray.data.DwInfoFlags = win32.NIIF_WARNING
	}
	copyUTF16(tray.data.SzInfoTitle[:], title)
	copyUTF16(tray.data.SzInfo[:], message)
	win32.Shell_NotifyIconW(win32.NIM_MODIFY, &tray.data)
	tray.data.UFlags = win32.NIF_MESSAGE | win32.NIF_ICON | win32.NIF_TIP
}

func (tray *trayIcon) showMenu() {
	menu, menuErr := win32.CreatePopupMenu()
	if menu == 0 {
		tray.app.logger.Printf("create tray menu: %v", menuErr)
		return
	}
	defer win32.DestroyMenu(menu)

	state := tray.app.state
	switch state {
	case stateSelecting:
		appendMenu(menu, win32.MF_STRING, commandLockRegion, tray.app.text.cancelSelection)
	case stateLocked:
		appendMenu(menu, win32.MF_STRING, commandUnlock, tray.app.text.unlock)
	default:
		appendMenu(menu, win32.MF_STRING, commandLockRegion, tray.app.text.lockRegion)
	}
	appendMenu(menu, win32.MF_SEPARATOR, 0, "")

	startFlags := win32.MF_STRING
	if isStartAtLoginEnabled() {
		startFlags |= win32.MF_CHECKED
	}
	appendMenu(menu, startFlags, commandStartAtLogin, tray.app.text.startAtLogin)
	appendMenu(menu, win32.MF_SEPARATOR, 0, "")
	appendMenu(menu, win32.MF_STRING, commandAbout, tray.app.text.about)
	appendMenu(menu, win32.MF_STRING, commandExit, tray.app.text.exit)

	var point win32.POINT
	if ok, cursorErr := win32.GetCursorPos(&point); ok == win32.FALSE {
		tray.app.logger.Printf("get cursor position: %v", cursorErr)
		return
	}
	win32.SetForegroundWindow(tray.app.controllerWindow)
	selected, trackErr := win32.TrackPopupMenu(
		menu,
		win32.TPM_RETURNCMD|win32.TPM_RIGHTBUTTON,
		point.X,
		point.Y,
		0,
		tray.app.controllerWindow,
		nil,
	)
	if trackErr != win32.NO_ERROR {
		tray.app.logger.Printf("track tray menu: %v", trackErr)
	}
	if selected != 0 {
		tray.app.handleCommand(uint32(selected))
	}
	win32.PostMessageW(tray.app.controllerWindow, win32.WM_NULL, 0, 0)
}

func appendMenu(menu win32.HMENU, flags win32.MENU_ITEM_FLAGS, command uint32, label string) {
	win32.AppendMenuW(menu, flags, uintptr(command), utf16Ptr(label))
}
