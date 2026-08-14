//go:build windows

package platform

import (
	"fmt"

	"github.com/william08190/region-lockpaw/windows/internal/geometry"
	"github.com/zzl/go-win32api/v2/win32"
)

const maskOpacity = 244

type maskController struct {
	app             *App
	allowed         geometry.Rect
	hasAllowed      bool
	windows         []*maskWindow
	backgroundBrush win32.HBRUSH
	accentBrush     win32.HBRUSH
	buttonBrush     win32.HBRUSH
	buttonEdgeBrush win32.HBRUSH
}

type maskWindow struct {
	owner  *maskController
	handle win32.HWND
	frame  geometry.Rect
}

func newMaskController(app *App, monitors []monitor, allowed geometry.Rect) (*maskController, error) {
	if !containsAllowedRegion(monitorBounds(monitors), allowed) {
		return nil, fmt.Errorf("the selected region is no longer inside an active display")
	}
	return createMaskController(app, monitors, allowed)
}

func newMaskControllerForDisplayChange(app *App, monitors []monitor, allowed geometry.Rect) (*maskController, bool, error) {
	effectiveAllowed, regionPreserved := displayChangeMaskRegion(monitorBounds(monitors), allowed)
	controller, err := createMaskController(app, monitors, effectiveAllowed)
	return controller, regionPreserved, err
}

func monitorBounds(monitors []monitor) []geometry.Rect {
	bounds := make([]geometry.Rect, 0, len(monitors))
	for _, display := range monitors {
		bounds = append(bounds, display.bounds)
	}
	return bounds
}

func createMaskController(app *App, monitors []monitor, allowed geometry.Rect) (*maskController, error) {
	controller := &maskController{
		app:             app,
		allowed:         allowed,
		hasAllowed:      !allowed.Empty(),
		backgroundBrush: win32.CreateSolidBrush(rgb(9, 12, 17)),
		accentBrush:     win32.CreateSolidBrush(rgb(0, 212, 170)),
		buttonBrush:     win32.CreateSolidBrush(rgb(34, 40, 49)),
		buttonEdgeBrush: win32.CreateSolidBrush(rgb(103, 113, 126)),
	}
	if controller.backgroundBrush == 0 || controller.accentBrush == 0 || controller.buttonBrush == 0 || controller.buttonEdgeBrush == 0 {
		controller.destroy()
		return nil, fmt.Errorf("create mask drawing resources")
	}
	for _, display := range monitors {
		for _, frame := range geometry.MaskFrames(display.bounds, allowed) {
			window, createErr := win32.CreateWindowExW(
				win32.WS_EX_TOPMOST|win32.WS_EX_TOOLWINDOW|win32.WS_EX_LAYERED|win32.WS_EX_NOACTIVATE,
				utf16Ptr(maskClassName),
				utf16Ptr("Region Lockpaw Mask"),
				win32.WS_POPUP,
				frame.Left,
				frame.Top,
				frame.Width(),
				frame.Height(),
				0, 0, app.instance, nil,
			)
			if window == 0 {
				controller.destroy()
				return nil, fmt.Errorf("create mask window: %w", createErr)
			}
			item := &maskWindow{owner: controller, handle: window, frame: frame}
			controller.windows = append(controller.windows, item)
			attachHandler(window, item)
			if ok, layerErr := win32.SetLayeredWindowAttributes(window, 0, maskOpacity, win32.LWA_ALPHA); ok == win32.FALSE {
				controller.destroy()
				return nil, fmt.Errorf("configure mask transparency: %w", layerErr)
			}
		}
	}
	if len(controller.windows) == 0 {
		controller.destroy()
		return nil, fmt.Errorf("the selected region covers every active display")
	}
	return controller, nil
}

func (controller *maskController) show() error {
	for _, window := range controller.windows {
		if ok, windowErr := win32.SetWindowPos(
			window.handle,
			win32.HWND_TOPMOST,
			window.frame.Left,
			window.frame.Top,
			window.frame.Width(),
			window.frame.Height(),
			win32.SWP_SHOWWINDOW|win32.SWP_NOACTIVATE,
		); ok == win32.FALSE {
			return fmt.Errorf("show mask window: %w", windowErr)
		}
		win32.ShowWindow(window.handle, win32.SW_SHOWNOACTIVATE)
		win32.UpdateWindow(window.handle)
	}
	return nil
}

func (controller *maskController) refreshTopmost() {
	for _, window := range controller.windows {
		win32.SetWindowPos(
			window.handle,
			win32.HWND_TOPMOST,
			0, 0, 0, 0,
			win32.SWP_NOMOVE|win32.SWP_NOSIZE|win32.SWP_NOACTIVATE,
		)
	}
}

func (controller *maskController) destroy() {
	for _, window := range controller.windows {
		if window.handle != 0 {
			detachHandler(window.handle)
			win32.DestroyWindow(window.handle)
			window.handle = 0
		}
	}
	controller.windows = nil
	if controller.backgroundBrush != 0 {
		win32.DeleteObject(win32.HGDIOBJ(controller.backgroundBrush))
		controller.backgroundBrush = 0
	}
	if controller.accentBrush != 0 {
		win32.DeleteObject(win32.HGDIOBJ(controller.accentBrush))
		controller.accentBrush = 0
	}
	if controller.buttonBrush != 0 {
		win32.DeleteObject(win32.HGDIOBJ(controller.buttonBrush))
		controller.buttonBrush = 0
	}
	if controller.buttonEdgeBrush != 0 {
		win32.DeleteObject(win32.HGDIOBJ(controller.buttonEdgeBrush))
		controller.buttonEdgeBrush = 0
	}
}

func (window *maskWindow) handleMessage(hwnd win32.HWND, message uint32, _ win32.WPARAM, _ win32.LPARAM) (win32.LRESULT, bool) {
	switch message {
	case win32.WM_LBUTTONUP:
		window.owner.app.requestSecureUnlock()
		return 0, true
	case win32.WM_MOUSEACTIVATE:
		return win32.LRESULT(win32.MA_NOACTIVATE), true
	case win32.WM_SETCURSOR:
		cursor, _ := win32.LoadCursorW(0, win32.IDC_HAND)
		win32.SetCursor(cursor)
		return 1, true
	case win32.WM_ERASEBKGND:
		return 1, true
	case win32.WM_PAINT:
		window.paint(hwnd)
		return 0, true
	}
	return 0, false
}

func (window *maskWindow) paint(hwnd win32.HWND) {
	var paint win32.PAINTSTRUCT
	deviceContext := win32.BeginPaint(hwnd, &paint)
	if deviceContext == 0 {
		return
	}
	defer win32.EndPaint(hwnd, &paint)

	var client win32.RECT
	win32.GetClientRect(hwnd, &client)
	win32.FillRect(deviceContext, &client, window.owner.backgroundBrush)

	const borderWidth int32 = 2
	allowed := window.owner.allowed
	frame := window.frame
	var edge win32.RECT
	if window.owner.hasAllowed {
		switch {
		case frame.Bottom == allowed.Top && frame.Left <= allowed.Left && frame.Right >= allowed.Right:
			edge = win32.RECT{Left: max(0, allowed.Left-frame.Left), Top: max(0, client.Bottom-borderWidth), Right: min(client.Right, allowed.Right-frame.Left), Bottom: client.Bottom}
		case frame.Top == allowed.Bottom && frame.Left <= allowed.Left && frame.Right >= allowed.Right:
			edge = win32.RECT{Left: max(0, allowed.Left-frame.Left), Top: 0, Right: min(client.Right, allowed.Right-frame.Left), Bottom: borderWidth}
		case frame.Right == allowed.Left && frame.Top == allowed.Top:
			edge = win32.RECT{Left: max(0, client.Right-borderWidth), Top: 0, Right: client.Right, Bottom: client.Bottom}
		case frame.Left == allowed.Right && frame.Top == allowed.Top:
			edge = win32.RECT{Left: 0, Top: 0, Right: borderWidth, Bottom: client.Bottom}
		}
	}
	if edge.Right > edge.Left && edge.Bottom > edge.Top {
		win32.FillRect(deviceContext, &edge, window.owner.accentBrush)
	}

	if client.Right-client.Left >= 150 && client.Bottom-client.Top >= 90 {
		const buttonWidth int32 = 104
		const buttonHeight int32 = 34
		left := (client.Right - buttonWidth) / 2
		top := (client.Bottom - buttonHeight) / 2
		button := win32.RECT{Left: left, Top: top, Right: left + buttonWidth, Bottom: top + buttonHeight}
		win32.FillRect(deviceContext, &button, window.owner.buttonBrush)
		win32.FrameRect(deviceContext, &button, window.owner.buttonEdgeBrush)
		win32.SetBkMode(deviceContext, win32.TRANSPARENT)
		win32.SetTextColor(deviceContext, rgb(242, 246, 248))
		font := win32.GetStockObject(win32.DEFAULT_GUI_FONT)
		previousFont := win32.SelectObject(deviceContext, font)
		win32.DrawTextW(deviceContext, utf16Ptr(window.owner.app.text.unlockButton), -1, &button, win32.DT_CENTER|win32.DT_VCENTER|win32.DT_SINGLELINE|win32.DT_NOPREFIX)
		win32.SelectObject(deviceContext, previousFont)
	}
}
