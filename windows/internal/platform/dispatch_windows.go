//go:build windows

package platform

import (
	"sync"

	"github.com/zzl/go-win32api/v2/win32"
	"golang.org/x/sys/windows"
)

type windowHandler interface {
	handleMessage(hwnd win32.HWND, message uint32, wParam win32.WPARAM, lParam win32.LPARAM) (win32.LRESULT, bool)
}

var (
	windowProcCallback = win32.WNDPROC(windows.NewCallback(dispatchWindowMessage))
	handlersMu         sync.RWMutex
	handlers           = make(map[win32.HWND]windowHandler)
)

func dispatchWindowMessage(hwnd uintptr, message uint32, wParam, lParam uintptr) uintptr {
	handlersMu.RLock()
	handler := handlers[win32.HWND(hwnd)]
	handlersMu.RUnlock()
	if handler != nil {
		if result, handled := handler.handleMessage(
			win32.HWND(hwnd),
			message,
			win32.WPARAM(wParam),
			win32.LPARAM(lParam),
		); handled {
			return uintptr(result)
		}
	}
	return uintptr(win32.DefWindowProcW(win32.HWND(hwnd), message, win32.WPARAM(wParam), win32.LPARAM(lParam)))
}

func attachHandler(hwnd win32.HWND, handler windowHandler) {
	handlersMu.Lock()
	handlers[hwnd] = handler
	handlersMu.Unlock()
}

func detachHandler(hwnd win32.HWND) {
	handlersMu.Lock()
	delete(handlers, hwnd)
	handlersMu.Unlock()
}
