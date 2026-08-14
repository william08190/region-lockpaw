//go:build windows

package platform

import (
	"fmt"
	"unsafe"

	"github.com/william08190/region-lockpaw/windows/internal/geometry"
	"github.com/zzl/go-win32api/v2/win32"
	"golang.org/x/sys/windows"
)

const (
	controllerClassName = "RegionLockpaw.Controller"
	selectorClassName   = "RegionLockpaw.Selector"
	maskClassName       = "RegionLockpaw.Mask"
	controllerTitle     = "Region Lockpaw Controller"

	wmTray              = win32.WM_APP + 1
	wmExternalCommand   = win32.WM_APP + 2
	wtsNotifyForSession = 0

	commandActivate      = 1
	commandLockRegion    = 1001
	commandUnlock        = 1002
	commandStartAtLogin  = 1003
	commandAbout         = 1004
	commandExit          = 1005
	hotkeyToggle         = 1
	topmostRefreshTimer  = 1
	displayRefreshTimer  = 2
	displayRefreshDelay  = 300
	minimumSelectionSize = 64
)

var (
	wtsapi32                             = windows.NewLazySystemDLL("wtsapi32.dll")
	procWTSRegisterSessionNotification   = wtsapi32.NewProc("WTSRegisterSessionNotification")
	procWTSUnRegisterSessionNotification = wtsapi32.NewProc("WTSUnRegisterSessionNotification")
)

type monitor struct {
	bounds  geometry.Rect
	primary bool
}

func utf16Ptr(value string) win32.PWSTR {
	ptr, err := windows.UTF16PtrFromString(value)
	if err != nil {
		panic(err)
	}
	return ptr
}

func copyUTF16(destination []uint16, value string) {
	clear(destination)
	if len(destination) == 0 {
		return
	}
	encoded := windows.StringToUTF16(value)
	copy(destination, encoded[:min(len(encoded), len(destination))])
	destination[len(destination)-1] = 0
}

func registerWindowClass(instance win32.HINSTANCE, name string, cursor win32.HCURSOR, icon win32.HICON) error {
	class := win32.WNDCLASSEXW{
		CbSize:        uint32(unsafe.Sizeof(win32.WNDCLASSEXW{})),
		Style:         win32.CS_HREDRAW | win32.CS_VREDRAW,
		LpfnWndProc:   windowProcCallback,
		HInstance:     instance,
		HIcon:         icon,
		HCursor:       cursor,
		LpszClassName: utf16Ptr(name),
		HIconSm:       icon,
	}
	atom, winErr := win32.RegisterClassExW(&class)
	if atom == 0 && winErr != win32.ERROR_CLASS_ALREADY_EXISTS {
		return fmt.Errorf("register window class %q: %w", name, winErr)
	}
	return nil
}

func enumerateMonitors() ([]monitor, error) {
	monitors := make([]monitor, 0, 2)
	callback := windows.NewCallback(func(hMonitor, _ uintptr, rect *win32.RECT, _ uintptr) uintptr {
		if rect == nil {
			return 1
		}
		info := win32.MONITORINFO{CbSize: uint32(unsafe.Sizeof(win32.MONITORINFO{}))}
		win32.GetMonitorInfoW(win32.HMONITOR(hMonitor), &info)
		monitors = append(monitors, monitor{
			bounds: geometry.Rect{
				Left:   rect.Left,
				Top:    rect.Top,
				Right:  rect.Right,
				Bottom: rect.Bottom,
			},
			primary: info.DwFlags&win32.MONITORINFOF_PRIMARY != 0,
		})
		return 1
	})

	if win32.EnumDisplayMonitors(0, nil, win32.MONITORENUMPROC(callback), 0) == win32.FALSE || len(monitors) == 0 {
		return nil, fmt.Errorf("enumerate displays: no active displays")
	}
	for index, item := range monitors {
		if item.primary && index != 0 {
			monitors[0], monitors[index] = monitors[index], monitors[0]
			break
		}
	}
	return monitors, nil
}

func registerSessionNotifications(hwnd win32.HWND) error {
	result, _, callErr := procWTSRegisterSessionNotification.Call(uintptr(hwnd), wtsNotifyForSession)
	if result == 0 {
		return fmt.Errorf("register session notifications: %w", callErr)
	}
	return nil
}

func unregisterSessionNotifications(hwnd win32.HWND) {
	if hwnd != 0 {
		procWTSUnRegisterSessionNotification.Call(uintptr(hwnd))
	}
}

func rgb(red, green, blue byte) win32.COLORREF {
	return win32.RGB(red, green, blue)
}
