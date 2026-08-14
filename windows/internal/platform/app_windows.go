//go:build windows

package platform

import (
	"fmt"
	"log"
	"runtime"
	"slices"

	"github.com/william08190/region-lockpaw/windows/internal/geometry"
	"github.com/zzl/go-win32api/v2/win32"
)

type appState uint8

const (
	stateIdle appState = iota
	stateSelecting
	stateLocked
)

type App struct {
	version               string
	text                  localizedText
	logger                *log.Logger
	instance              win32.HINSTANCE
	controllerWindow      win32.HWND
	icon                  win32.HICON
	tray                  *trayIcon
	selector              *selectorController
	masks                 *maskController
	state                 appState
	lockedRegion          geometry.Rect
	previousForeground    win32.HWND
	sessionNotifications  bool
	secureUnlock          secureUnlockFlow
	powerRequestActive    bool
	taskbarCreatedMessage uint32
	hotkeyRegistered      bool
	cleanedUp             bool
}

func Run(version string, arguments []string) error {
	runtime.LockOSThread()
	defer runtime.UnlockOSThread()

	logger, closeLogger := openAppLogger()
	defer closeLogger()

	win32.SetProcessDpiAwarenessContext(win32.DPI_AWARENESS_CONTEXT_PER_MONITOR_AWARE_V2)

	mutex, mutexErr := win32.CreateMutexW(nil, win32.FALSE, utf16Ptr(`Local\RegionLockpaw.SingleInstance`))
	if mutex == 0 {
		return fmt.Errorf("create single-instance mutex: %w", mutexErr)
	}
	defer win32.CloseHandle(mutex)
	if mutexErr == win32.ERROR_ALREADY_EXISTS {
		activateExistingInstance(arguments)
		return nil
	}

	module, moduleErr := win32.GetModuleHandleW(nil)
	if module == 0 {
		return fmt.Errorf("get application module: %w", moduleErr)
	}
	app := &App{
		version:  version,
		text:     loadLocalizedText(version),
		logger:   logger,
		instance: win32.HINSTANCE(module),
		state:    stateIdle,
	}
	if err := app.initialize(); err != nil {
		app.cleanup()
		return err
	}
	defer app.cleanup()
	if slices.Contains(arguments, "--self-test") {
		app.shutdown()
		return nil
	}

	if slices.Contains(arguments, "--lock-region") {
		win32.PostMessageW(app.controllerWindow, wmExternalCommand, commandLockRegion, 0)
	} else if !slices.Contains(arguments, "--background") {
		app.tray.showBalloon(app.text.ready, app.text.readyDetail, false)
	}
	return app.messageLoop()
}

func ShowFatalError(err error) {
	text := loadLocalizedText("unknown")
	detail := text.unexpectedError
	if err != nil {
		detail += "\n\n" + err.Error()
	}
	win32.MessageBoxW(0, utf16Ptr(detail), utf16Ptr("Region Lockpaw"), win32.MB_OK|win32.MB_ICONERROR|win32.MB_TOPMOST)
}

func (app *App) initialize() error {
	icon, _ := win32.LoadIconW(app.instance, win32.MAKEINTRESOURCE(1))
	if icon == 0 {
		icon, _ = win32.LoadIconW(0, win32.IDI_APPLICATION)
	}
	app.icon = icon
	arrow, _ := win32.LoadCursorW(0, win32.IDC_ARROW)
	cross, _ := win32.LoadCursorW(0, win32.IDC_CROSS)
	hand, _ := win32.LoadCursorW(0, win32.IDC_HAND)

	if err := registerWindowClass(app.instance, controllerClassName, arrow, icon); err != nil {
		return err
	}
	if err := registerWindowClass(app.instance, selectorClassName, cross, icon); err != nil {
		return err
	}
	if err := registerWindowClass(app.instance, maskClassName, hand, icon); err != nil {
		return err
	}

	window, windowErr := win32.CreateWindowExW(
		win32.WS_EX_TOOLWINDOW,
		utf16Ptr(controllerClassName),
		utf16Ptr(controllerTitle),
		win32.WS_OVERLAPPED,
		0, 0, 0, 0,
		0, 0, app.instance, nil,
	)
	if window == 0 {
		return fmt.Errorf("create controller window: %w", windowErr)
	}
	app.controllerWindow = window
	attachHandler(window, app)

	taskbarMessage, taskbarErr := win32.RegisterWindowMessageW(utf16Ptr("TaskbarCreated"))
	if taskbarMessage == 0 {
		app.logger.Printf("register TaskbarCreated message: %v", taskbarErr)
	}
	app.taskbarCreatedMessage = taskbarMessage

	app.tray = newTrayIcon(app)
	if err := app.tray.add(); err != nil {
		return err
	}
	if ok, hotkeyErr := win32.RegisterHotKey(
		app.controllerWindow,
		hotkeyToggle,
		win32.MOD_CONTROL|win32.MOD_ALT|win32.MOD_NOREPEAT,
		uint32('L'),
	); ok == win32.FALSE {
		app.logger.Printf("register global hotkey: %v", hotkeyErr)
		app.tray.showBalloon("Region Lockpaw", app.text.registrationFailed, true)
	} else {
		app.hotkeyRegistered = true
	}
	if err := registerSessionNotifications(app.controllerWindow); err != nil {
		app.logger.Printf("%v", err)
	} else {
		app.sessionNotifications = true
	}
	if timer, timerErr := win32.SetTimer(app.controllerWindow, topmostRefreshTimer, 2000, 0); timer == 0 {
		app.logger.Printf("create topmost refresh timer: %v", timerErr)
	}
	app.logger.Printf("started version=%s", app.version)
	return nil
}

func (app *App) messageLoop() error {
	var message win32.MSG
	for {
		result, messageErr := win32.GetMessageW(&message, 0, 0, 0)
		switch {
		case result > 0:
			win32.TranslateMessage(&message)
			win32.DispatchMessageW(&message)
		case result == 0:
			return nil
		default:
			return fmt.Errorf("read Windows message: %w", messageErr)
		}
	}
}

func (app *App) handleMessage(_ win32.HWND, message uint32, wParam win32.WPARAM, lParam win32.LPARAM) (win32.LRESULT, bool) {
	if message == app.taskbarCreatedMessage && message != 0 {
		if err := app.tray.add(); err != nil {
			app.logger.Printf("restore notification-area icon: %v", err)
		}
		return 0, true
	}

	switch message {
	case wmTray:
		switch uint32(lParam) {
		case win32.WM_LBUTTONDBLCLK:
			app.toggle()
		case win32.WM_RBUTTONUP, win32.WM_CONTEXTMENU:
			app.tray.showMenu()
		}
		return 0, true
	case win32.WM_HOTKEY:
		if int32(wParam) == hotkeyToggle {
			app.toggle()
		}
		return 0, true
	case wmExternalCommand:
		app.handleCommand(uint32(wParam))
		return 0, true
	case win32.WM_DISPLAYCHANGE:
		app.handleDisplayChange()
		return 0, true
	case win32.WM_WTSSESSION_CHANGE:
		app.handleSessionChange(uint32(wParam))
		return 0, true
	case win32.WM_TIMER:
		switch uintptr(wParam) {
		case topmostRefreshTimer:
			if app.masks != nil {
				app.masks.refreshTopmost()
			}
			if app.selector != nil {
				app.selector.refreshTopmost()
			}
		case displayRefreshTimer:
			win32.KillTimer(app.controllerWindow, displayRefreshTimer)
			app.refreshMasksForDisplayChange()
		}
		return 0, true
	case win32.WM_CLOSE:
		app.shutdown()
		return 0, true
	case win32.WM_DESTROY:
		win32.PostQuitMessage(0)
		return 0, true
	}
	return 0, false
}

func (app *App) handleCommand(command uint32) {
	switch command {
	case commandActivate:
		app.tray.showBalloon(app.text.ready, app.text.readyDetail, false)
	case commandLockRegion:
		if app.state == stateSelecting {
			app.cancelSelection()
		} else if app.state == stateIdle {
			app.beginSelection()
		}
	case commandUnlock:
		if app.state == stateLocked {
			app.unlock()
		}
	case commandStartAtLogin:
		if err := setStartAtLogin(!isStartAtLoginEnabled()); err != nil {
			app.logger.Printf("set start at sign-in: %v", err)
			app.tray.showBalloon("Region Lockpaw", app.text.startAtLoginError, true)
		}
	case commandAbout:
		body := app.text.aboutBody + "\n\n" + app.text.visualPrivacyNotice
		win32.MessageBoxW(app.controllerWindow, utf16Ptr(body), utf16Ptr("Region Lockpaw"), win32.MB_OK|win32.MB_ICONINFORMATION|win32.MB_TOPMOST)
	case commandExit:
		app.shutdown()
	}
}

func (app *App) toggle() {
	switch app.state {
	case stateIdle:
		app.beginSelection()
	case stateSelecting:
		app.cancelSelection()
	case stateLocked:
		app.unlock()
	}
}

func (app *App) beginSelection() {
	monitors, err := enumerateMonitors()
	if err != nil {
		app.logger.Printf("start selection: %v", err)
		app.tray.showBalloon("Region Lockpaw", err.Error(), true)
		return
	}
	app.previousForeground = win32.GetForegroundWindow()
	selector, err := newSelectorController(app, monitors)
	if err != nil {
		app.logger.Printf("start selection: %v", err)
		app.restorePreviousForeground()
		app.tray.showBalloon("Region Lockpaw", err.Error(), true)
		return
	}
	app.selector = selector
	app.state = stateSelecting
	if err := selector.show(); err != nil {
		app.logger.Printf("show selection: %v", err)
		app.destroySelector()
		app.state = stateIdle
		app.restorePreviousForeground()
		app.tray.showBalloon("Region Lockpaw", err.Error(), true)
	}
}

func (app *App) completeSelection(rect geometry.Rect) {
	if app.state != stateSelecting {
		return
	}
	app.destroySelector()
	monitors, err := enumerateMonitors()
	if err != nil {
		app.logger.Printf("create region mask: %v", err)
		app.state = stateIdle
		app.restorePreviousForeground()
		return
	}
	masks, err := newMaskController(app, monitors, rect)
	if err != nil {
		app.logger.Printf("create region mask: %v", err)
		app.state = stateIdle
		app.restorePreviousForeground()
		app.tray.showBalloon("Region Lockpaw", err.Error(), true)
		return
	}
	app.masks = masks
	app.lockedRegion = rect
	if err := masks.show(); err != nil {
		app.logger.Printf("show region mask: %v", err)
		masks.destroy()
		app.masks = nil
		app.lockedRegion = geometry.Rect{}
		app.state = stateIdle
		app.restorePreviousForeground()
		app.tray.showBalloon("Region Lockpaw", err.Error(), true)
		return
	}
	app.state = stateLocked
	app.secureUnlock.reset()
	if previous := win32.SetThreadExecutionState(win32.ES_CONTINUOUS | win32.ES_DISPLAY_REQUIRED | win32.ES_SYSTEM_REQUIRED); previous == 0 {
		app.logger.Printf("prevent display sleep: Windows rejected the execution-state request")
	} else {
		app.powerRequestActive = true
	}
	app.restorePreviousForeground()
	app.logger.Printf("region locked left=%d top=%d right=%d bottom=%d", rect.Left, rect.Top, rect.Right, rect.Bottom)
}

func (app *App) cancelSelection() {
	if app.state != stateSelecting {
		return
	}
	app.destroySelector()
	app.state = stateIdle
	app.restorePreviousForeground()
}

func (app *App) destroySelector() {
	if app.selector != nil {
		app.selector.destroy()
		app.selector = nil
	}
}

func (app *App) restorePreviousForeground() {
	if app.previousForeground != 0 {
		win32.SetForegroundWindow(app.previousForeground)
		app.previousForeground = 0
	}
}

func (app *App) unlock() {
	if app.state != stateLocked {
		return
	}
	if app.controllerWindow != 0 {
		win32.KillTimer(app.controllerWindow, displayRefreshTimer)
	}
	if app.masks != nil {
		app.masks.destroy()
		app.masks = nil
	}
	app.lockedRegion = geometry.Rect{}
	app.secureUnlock.reset()
	if app.powerRequestActive {
		win32.SetThreadExecutionState(win32.ES_CONTINUOUS)
		app.powerRequestActive = false
	}
	app.state = stateIdle
	app.logger.Printf("region unlocked")
}

func (app *App) requestSecureUnlock() {
	if app.state != stateLocked {
		return
	}
	if !app.sessionNotifications {
		app.logger.Printf("secure unlock unavailable: session notifications are not registered")
		win32.MessageBoxW(app.controllerWindow, utf16Ptr(app.text.unlockFailed), utf16Ptr(app.text.unlockTitle), win32.MB_OK|win32.MB_ICONERROR|win32.MB_TOPMOST)
		return
	}
	foreground := win32.GetForegroundWindow()
	result, _ := win32.MessageBoxW(
		app.controllerWindow,
		utf16Ptr(app.text.unlockPrompt),
		utf16Ptr(app.text.unlockTitle),
		win32.MB_YESNO|win32.MB_ICONINFORMATION|win32.MB_TOPMOST|win32.MB_SETFOREGROUND,
	)
	if result != win32.IDYES || app.state != stateLocked || app.cleanedUp {
		if foreground != 0 {
			win32.SetForegroundWindow(foreground)
		}
		return
	}
	app.secureUnlock.begin()
	if ok, lockErr := win32.LockWorkStation(); ok == win32.FALSE {
		app.secureUnlock.reset()
		app.logger.Printf("open Windows secure sign-in: %v", lockErr)
		win32.MessageBoxW(app.controllerWindow, utf16Ptr(app.text.unlockFailed), utf16Ptr(app.text.unlockTitle), win32.MB_OK|win32.MB_ICONERROR|win32.MB_TOPMOST)
		if foreground != 0 {
			win32.SetForegroundWindow(foreground)
		}
	}
}

func (app *App) handleSessionChange(event uint32) {
	if app.state != stateLocked {
		return
	}
	switch event {
	case win32.WTS_SESSION_LOCK:
		if app.secureUnlock.observeSessionLock() {
			app.logger.Printf("Windows session locked for requested secure unlock")
		} else {
			app.logger.Printf("Windows session locked externally; preserving region lock")
		}
	case win32.WTS_SESSION_UNLOCK:
		if app.secureUnlock.observeSessionUnlock() {
			app.unlock()
		} else {
			if app.masks != nil {
				app.masks.refreshTopmost()
			}
			app.logger.Printf("Windows session unlocked without a requested secure unlock; preserving region lock")
		}
	}
}

func (app *App) handleDisplayChange() {
	if app.state == stateSelecting {
		app.cancelSelection()
		return
	}
	if app.state == stateLocked {
		if timer, timerErr := win32.SetTimer(app.controllerWindow, displayRefreshTimer, displayRefreshDelay, 0); timer == 0 {
			app.logger.Printf("schedule display-change mask refresh: %v", timerErr)
			app.refreshMasksForDisplayChange()
		}
	}
}

func (app *App) refreshMasksForDisplayChange() {
	if app.state != stateLocked || app.lockedRegion.Empty() {
		return
	}
	monitors, err := enumerateMonitors()
	if err != nil {
		app.logger.Printf("refresh masks after display change: %v; preserving existing masks", err)
		return
	}
	replacement, regionPreserved, err := newMaskControllerForDisplayChange(app, monitors, app.lockedRegion)
	if err != nil {
		app.logger.Printf("refresh masks after display change: %v; preserving existing masks", err)
		return
	}
	if err := replacement.show(); err != nil {
		replacement.destroy()
		app.logger.Printf("show refreshed masks after display change: %v; preserving existing masks", err)
		return
	}
	previous := app.masks
	app.masks = replacement
	if previous != nil {
		previous.destroy()
	}
	if regionPreserved {
		app.logger.Printf("display layout changed; rebuilt masks while preserving the selected region")
		return
	}
	app.logger.Printf("display layout changed; selected region unavailable, masking all active displays")
	app.tray.showBalloon("Region Lockpaw", app.text.displayChanged, true)
}

func (app *App) shutdown() {
	app.cleanup()
	if app.controllerWindow != 0 {
		window := app.controllerWindow
		app.controllerWindow = 0
		detachHandler(window)
		win32.DestroyWindow(window)
	}
	win32.PostQuitMessage(0)
}

func (app *App) cleanup() {
	if app.cleanedUp {
		return
	}
	app.cleanedUp = true
	app.destroySelector()
	app.restorePreviousForeground()
	if app.masks != nil {
		app.masks.destroy()
		app.masks = nil
	}
	app.state = stateIdle
	app.lockedRegion = geometry.Rect{}
	app.secureUnlock.reset()
	if app.powerRequestActive {
		win32.SetThreadExecutionState(win32.ES_CONTINUOUS)
		app.powerRequestActive = false
	}
	if app.tray != nil {
		app.tray.remove()
	}
	if app.controllerWindow != 0 {
		win32.KillTimer(app.controllerWindow, topmostRefreshTimer)
		win32.KillTimer(app.controllerWindow, displayRefreshTimer)
		if app.hotkeyRegistered {
			win32.UnregisterHotKey(app.controllerWindow, hotkeyToggle)
		}
		if app.sessionNotifications {
			unregisterSessionNotifications(app.controllerWindow)
		}
	}
	app.logger.Printf("stopped")
}

func activateExistingInstance(arguments []string) {
	window, _ := win32.FindWindowW(utf16Ptr(controllerClassName), utf16Ptr(controllerTitle))
	if window == 0 {
		return
	}
	command := uintptr(commandActivate)
	if slices.Contains(arguments, "--lock-region") {
		command = commandLockRegion
	}
	win32.PostMessageW(window, wmExternalCommand, win32.WPARAM(command), 0)
}
