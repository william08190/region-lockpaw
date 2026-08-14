//go:build windows

package platform

import (
	"fmt"

	"github.com/william08190/region-lockpaw/windows/internal/geometry"
	"github.com/zzl/go-win32api/v2/win32"
)

const selectionOpacity = 158

type selectorController struct {
	app             *App
	windows         []*selectorWindow
	active          *selectorWindow
	dragging        bool
	start           geometry.Point
	current         geometry.Point
	backgroundBrush win32.HBRUSH
	selectionBrush  win32.HBRUSH
	accentBrush     win32.HBRUSH
	badgeBrush      win32.HBRUSH
}

type selectorWindow struct {
	owner  *selectorController
	handle win32.HWND
	bounds geometry.Rect
}

func newSelectorController(app *App, monitors []monitor) (*selectorController, error) {
	controller := &selectorController{
		app:             app,
		backgroundBrush: win32.CreateSolidBrush(rgb(12, 15, 20)),
		selectionBrush:  win32.CreateSolidBrush(rgb(8, 70, 62)),
		accentBrush:     win32.CreateSolidBrush(rgb(0, 212, 170)),
		badgeBrush:      win32.CreateSolidBrush(rgb(15, 20, 27)),
	}
	if controller.backgroundBrush == 0 || controller.selectionBrush == 0 || controller.accentBrush == 0 || controller.badgeBrush == 0 {
		controller.destroy()
		return nil, fmt.Errorf("create selection drawing resources")
	}

	for _, display := range monitors {
		window, createErr := win32.CreateWindowExW(
			win32.WS_EX_TOPMOST|win32.WS_EX_TOOLWINDOW|win32.WS_EX_LAYERED,
			utf16Ptr(selectorClassName),
			utf16Ptr("Region Lockpaw Selection"),
			win32.WS_POPUP,
			display.bounds.Left,
			display.bounds.Top,
			display.bounds.Width(),
			display.bounds.Height(),
			0, 0, app.instance, nil,
		)
		if window == 0 {
			controller.destroy()
			return nil, fmt.Errorf("create selection window: %w", createErr)
		}
		item := &selectorWindow{owner: controller, handle: window, bounds: display.bounds}
		controller.windows = append(controller.windows, item)
		attachHandler(window, item)
		if ok, layerErr := win32.SetLayeredWindowAttributes(window, 0, selectionOpacity, win32.LWA_ALPHA); ok == win32.FALSE {
			controller.destroy()
			return nil, fmt.Errorf("configure selection transparency: %w", layerErr)
		}
	}
	if len(controller.windows) == 0 {
		controller.destroy()
		return nil, fmt.Errorf("no selection windows were created")
	}
	return controller, nil
}

func (controller *selectorController) show() error {
	for _, window := range controller.windows {
		if ok, windowErr := win32.SetWindowPos(
			window.handle,
			win32.HWND_TOPMOST,
			window.bounds.Left,
			window.bounds.Top,
			window.bounds.Width(),
			window.bounds.Height(),
			win32.SWP_SHOWWINDOW|win32.SWP_NOACTIVATE,
		); ok == win32.FALSE {
			return fmt.Errorf("show selection window: %w", windowErr)
		}
		win32.UpdateWindow(window.handle)
	}
	first := controller.windows[0]
	win32.SetForegroundWindow(first.handle)
	win32.SetFocus(first.handle)
	return nil
}

func (controller *selectorController) beginDrag(window *selectorWindow) {
	var cursor win32.POINT
	if ok, cursorErr := win32.GetCursorPos(&cursor); ok == win32.FALSE {
		controller.app.logger.Printf("read drag start: %v", cursorErr)
		return
	}
	point := geometry.ClampPoint(geometry.Point{X: cursor.X, Y: cursor.Y}, window.bounds)
	controller.active = window
	controller.dragging = true
	controller.start = point
	controller.current = point
	win32.SetCapture(window.handle)
	win32.InvalidateRect(window.handle, nil, win32.FALSE)
}

func (controller *selectorController) updateDrag(window *selectorWindow) {
	if !controller.dragging || controller.active != window {
		return
	}
	var cursor win32.POINT
	if ok, cursorErr := win32.GetCursorPos(&cursor); ok == win32.FALSE {
		controller.app.logger.Printf("read drag position: %v", cursorErr)
		return
	}
	controller.current = geometry.ClampPoint(geometry.Point{X: cursor.X, Y: cursor.Y}, window.bounds)
	win32.InvalidateRect(window.handle, nil, win32.FALSE)
}

func (controller *selectorController) finishDrag(window *selectorWindow) {
	if !controller.dragging || controller.active != window {
		return
	}
	controller.updateDrag(window)
	controller.dragging = false
	win32.ReleaseCapture()
	selected := geometry.NewRect(controller.start, controller.current)
	if selected.Width() < minimumSelectionSize || selected.Height() < minimumSelectionSize {
		controller.active = nil
		win32.MessageBeep(win32.MB_ICONWARNING)
		win32.InvalidateRect(window.handle, nil, win32.FALSE)
		return
	}
	controller.app.completeSelection(selected)
}

func (controller *selectorController) refreshTopmost() {
	for _, window := range controller.windows {
		win32.SetWindowPos(
			window.handle,
			win32.HWND_TOPMOST,
			0, 0, 0, 0,
			win32.SWP_NOMOVE|win32.SWP_NOSIZE|win32.SWP_NOACTIVATE,
		)
	}
}

func (controller *selectorController) destroy() {
	if controller.dragging {
		controller.dragging = false
		win32.ReleaseCapture()
	}
	for _, window := range controller.windows {
		if window.handle != 0 {
			detachHandler(window.handle)
			win32.DestroyWindow(window.handle)
			window.handle = 0
		}
	}
	controller.windows = nil
	for _, brush := range []win32.HBRUSH{
		controller.backgroundBrush,
		controller.selectionBrush,
		controller.accentBrush,
		controller.badgeBrush,
	} {
		if brush != 0 {
			win32.DeleteObject(win32.HGDIOBJ(brush))
		}
	}
}

func (window *selectorWindow) handleMessage(hwnd win32.HWND, message uint32, wParam win32.WPARAM, _ win32.LPARAM) (win32.LRESULT, bool) {
	switch message {
	case win32.WM_LBUTTONDOWN:
		window.owner.beginDrag(window)
		return 0, true
	case win32.WM_MOUSEMOVE:
		window.owner.updateDrag(window)
		return 0, true
	case win32.WM_LBUTTONUP:
		window.owner.finishDrag(window)
		return 0, true
	case win32.WM_RBUTTONUP:
		window.owner.app.cancelSelection()
		return 0, true
	case win32.WM_KEYDOWN:
		if uint32(wParam) == uint32(win32.VK_ESCAPE) {
			window.owner.app.cancelSelection()
			return 0, true
		}
	case win32.WM_SETCURSOR:
		cursor, _ := win32.LoadCursorW(0, win32.IDC_CROSS)
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

func (window *selectorWindow) paint(hwnd win32.HWND) {
	var paint win32.PAINTSTRUCT
	deviceContext := win32.BeginPaint(hwnd, &paint)
	if deviceContext == 0 {
		return
	}
	defer win32.EndPaint(hwnd, &paint)

	var client win32.RECT
	win32.GetClientRect(hwnd, &client)
	win32.FillRect(deviceContext, &client, window.owner.backgroundBrush)

	controller := window.owner
	if !controller.dragging || controller.active != window {
		return
	}
	selected := geometry.NewRect(controller.start, controller.current)
	local := win32.RECT{
		Left:   selected.Left - window.bounds.Left,
		Top:    selected.Top - window.bounds.Top,
		Right:  selected.Right - window.bounds.Left,
		Bottom: selected.Bottom - window.bounds.Top,
	}
	win32.FillRect(deviceContext, &local, controller.selectionBrush)
	for index := int32(0); index < 3; index++ {
		border := win32.RECT{Left: local.Left + index, Top: local.Top + index, Right: local.Right - index, Bottom: local.Bottom - index}
		win32.FrameRect(deviceContext, &border, controller.accentBrush)
	}

	label := fmt.Sprintf("%d x %d", selected.Width(), selected.Height())
	labelWidth := min(int32(132), max(int32(60), selected.Width()-8))
	labelHeight := int32(30)
	labelLeft := local.Left + (selected.Width()-labelWidth)/2
	labelTop := local.Top + 12
	if selected.Height() < labelHeight+24 {
		labelTop = local.Top
	}
	badge := win32.RECT{Left: labelLeft, Top: labelTop, Right: labelLeft + labelWidth, Bottom: labelTop + labelHeight}
	win32.FillRect(deviceContext, &badge, controller.badgeBrush)
	win32.SetBkMode(deviceContext, win32.TRANSPARENT)
	win32.SetTextColor(deviceContext, rgb(235, 247, 244))
	font := win32.GetStockObject(win32.DEFAULT_GUI_FONT)
	previousFont := win32.SelectObject(deviceContext, font)
	win32.DrawTextW(deviceContext, utf16Ptr(label), -1, &badge, win32.DT_CENTER|win32.DT_VCENTER|win32.DT_SINGLELINE|win32.DT_NOPREFIX)
	win32.SelectObject(deviceContext, previousFont)
}
