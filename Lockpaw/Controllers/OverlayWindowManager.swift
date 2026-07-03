import AppKit
import SwiftUI
import os.log

private let logger = Logger(subsystem: "com.eriknielsen.lockpaw", category: "OverlayWindow")

/// Borderless windows refuse key status by default; the primary overlay must be able
/// to become key so the app can be activated while locked — cursor concealment
/// (`NSCursor.setHiddenUntilMouseMoves`) only works while the app is active.
private final class OverlayWindow: NSWindow {
    override var canBecomeKey: Bool { true }
}

class OverlayWindowManager {
    private var windows: [NSWindow] = []
    private var screenObserver: Any?
    private var sessionObserver: Any?
    private var contentFactory: ((Int, Bool) -> AnyView)?
    private var regionAllowedFrame: NSRect?
    private var regionUnlockHandler: (() -> Void)?
    private var screenChangeWork: DispatchWorkItem?
    private var mouseMoveMonitors: [Any] = []
    private var cursorRehideTimer: Timer?

    private let shieldLevel = NSWindow.Level(rawValue: Int(CGShieldingWindowLevel()))

    @discardableResult
    func showOverlay(contentFactory factory: @escaping (Int, Bool) -> AnyView) -> Bool {
        dismissOverlay()
        contentFactory = factory
        regionAllowedFrame = nil
        regionUnlockHandler = nil
        createWindows()
        guard !windows.isEmpty else {
            logger.error("showOverlay failed — no windows created")
            return false
        }
        startObservingScreenChanges()
        startObservingSessionChanges()
        startCursorConcealment()
        return true
    }

    @discardableResult
    func showRegionOverlay(allowedFrame: NSRect, onUnlock: @escaping () -> Void) -> Bool {
        dismissOverlay()
        contentFactory = nil
        regionAllowedFrame = allowedFrame
        regionUnlockHandler = onUnlock
        createRegionWindows()
        guard !windows.isEmpty else {
            logger.error("showRegionOverlay failed - no mask windows created")
            regionAllowedFrame = nil
            regionUnlockHandler = nil
            return false
        }
        startObservingScreenChanges()
        startObservingSessionChanges()
        return true
    }

    func dismissOverlay(animated: Bool = false) {
        stopObservingScreenChanges()
        stopObservingSessionChanges()
        stopCursorConcealment()
        contentFactory = nil
        regionAllowedFrame = nil
        regionUnlockHandler = nil

        for window in windows {
            window.orderOut(nil)
            window.contentView = nil
            window.close()
        }
        windows.removeAll()
    }

    func allowSystemDialogs() {
        for window in windows { window.level = .statusBar }
    }

    func blockSystemDialogs() {
        for window in windows { window.level = shieldLevel }
    }

    private func createWindows() {
        guard let factory = contentFactory else {
            logger.error("No content factory to display in overlay")
            return
        }
        let screens = NSScreen.screens
        guard !screens.isEmpty else {
            logger.critical("No screens available — cannot create overlay")
            return
        }

        for (index, screen) in screens.enumerated() {
            let isPrimary = (index == 0)
            let content = factory(index, isPrimary)
            let frame = screen.frame
            logger.info("Creating overlay — screen: \(screen.localizedName), role: \(isPrimary ? "primary" : "ambient"), frame: \(frame.debugDescription), scale: \(screen.backingScaleFactor)")
            let window = OverlayWindow(
                contentRect: frame,
                styleMask: .borderless,
                backing: .buffered,
                defer: false
            )
            window.setFrame(frame, display: true)
            window.level = shieldLevel
            window.collectionBehavior = [.canJoinAllSpaces, .fullScreenAuxiliary, .stationary]
            window.isOpaque = false
            window.backgroundColor = .clear
            window.ignoresMouseEvents = !isPrimary
            window.hasShadow = false
            window.isReleasedWhenClosed = false

            // NSHostingView defaults to autoresizingMask=0 (no flex), which can cause
            // the SwiftUI content to not fill the window on external/scaled displays.
            let hostingView = NSHostingView(rootView: content)
            hostingView.autoresizingMask = [.width, .height]
            hostingView.frame = window.contentLayoutRect
            window.contentView = hostingView

            if hostingView.frame.size != frame.size {
                logger.warning("Content view size mismatch — expected \(frame.size.debugDescription), got \(hostingView.frame.size.debugDescription)")
            }

            window.orderFrontRegardless()

            windows.append(window)
        }
    }

    private func createRegionWindows() {
        guard let allowedFrame = regionAllowedFrame, let regionUnlockHandler else {
            logger.error("No region configuration to display overlay")
            return
        }

        let screens = NSScreen.screens
        guard !screens.isEmpty else {
            logger.critical("No screens available - cannot create region overlay")
            return
        }

        for screen in screens {
            let screenFrame = screen.frame
            let allowedOnScreen = allowedFrame.intersection(screenFrame)
            let masks = maskFrames(for: screenFrame, excluding: allowedOnScreen)

            logger.info("Creating region overlay - screen: \(screen.localizedName), masks: \(masks.count), allowed: \(allowedOnScreen.debugDescription)")

            for frame in masks where frame.width > 0 && frame.height > 0 {
                let window = makeRegionMaskWindow(frame: frame, screen: screen, onUnlock: regionUnlockHandler)
                window.orderFrontRegardless()
                windows.append(window)
            }
        }
    }

    private func makeRegionMaskWindow(frame: NSRect, screen: NSScreen, onUnlock: @escaping () -> Void) -> NSWindow {
        let window = OverlayWindow(
            contentRect: frame,
            styleMask: .borderless,
            backing: .buffered,
            defer: false
        )
        window.setFrame(frame, display: true)
        window.level = shieldLevel
        window.collectionBehavior = [.canJoinAllSpaces, .fullScreenAuxiliary, .stationary]
        window.isOpaque = false
        window.backgroundColor = .clear
        window.ignoresMouseEvents = false
        window.hasShadow = false
        window.isReleasedWhenClosed = false

        let hostingView = NSHostingView(rootView: RegionMaskView(onUnlock: onUnlock))
        hostingView.autoresizingMask = [.width, .height]
        hostingView.frame = NSRect(origin: .zero, size: frame.size)
        window.contentView = hostingView

        return window
    }

    private func maskFrames(for screenFrame: NSRect, excluding allowedFrame: NSRect) -> [NSRect] {
        guard !allowedFrame.isNull, allowedFrame.width > 0, allowedFrame.height > 0 else {
            return [screenFrame]
        }

        let top = NSRect(
            x: screenFrame.minX,
            y: allowedFrame.maxY,
            width: screenFrame.width,
            height: screenFrame.maxY - allowedFrame.maxY
        )
        let bottom = NSRect(
            x: screenFrame.minX,
            y: screenFrame.minY,
            width: screenFrame.width,
            height: allowedFrame.minY - screenFrame.minY
        )
        let left = NSRect(
            x: screenFrame.minX,
            y: allowedFrame.minY,
            width: allowedFrame.minX - screenFrame.minX,
            height: allowedFrame.height
        )
        let right = NSRect(
            x: allowedFrame.maxX,
            y: allowedFrame.minY,
            width: screenFrame.maxX - allowedFrame.maxX,
            height: allowedFrame.height
        )

        return [top, bottom, left, right].filter { $0.width > 0 && $0.height > 0 }
    }

    // MARK: - Cursor concealment

    /// Hide the pointer while locked, but never trap the user: the cursor reappears
    /// the moment the mouse moves (it's needed to reach the fallback-auth controls)
    /// and slips away again after a few seconds of stillness. NSCursor.hide() is
    /// deliberately avoided — an unbalanced hide would leave the pointer invisible
    /// while someone tries to click the unlock chevron.
    private func startCursorConcealment() {
        stopCursorConcealment()
        // setHiddenUntilMouseMoves only takes effect while the app is active — and
        // when locking via the global hotkey some other app is frontmost. Activate
        // and make the primary overlay key first, then hide on the next runloop turn
        // so the activation has landed.
        NSApp.activate(ignoringOtherApps: true)
        windows.first?.makeKey()
        NSCursor.setHiddenUntilMouseMoves(true)
        DispatchQueue.main.async {
            NSCursor.setHiddenUntilMouseMoves(true)
        }
        let onMove: () -> Void = { [weak self] in self?.scheduleCursorRehide() }
        if let global = NSEvent.addGlobalMonitorForEvents(matching: [.mouseMoved], handler: { _ in onMove() }) {
            mouseMoveMonitors.append(global)
        }
        if let local = NSEvent.addLocalMonitorForEvents(matching: [.mouseMoved], handler: { event in
            onMove()
            return event
        }) {
            mouseMoveMonitors.append(local)
        }
    }

    private func scheduleCursorRehide() {
        cursorRehideTimer?.invalidate()
        cursorRehideTimer = Timer.scheduledTimer(withTimeInterval: Constants.Timing.cursorIdleHide, repeats: false) { _ in
            NSCursor.setHiddenUntilMouseMoves(true)
        }
    }

    private func stopCursorConcealment() {
        cursorRehideTimer?.invalidate()
        cursorRehideTimer = nil
        for monitor in mouseMoveMonitors { NSEvent.removeMonitor(monitor) }
        mouseMoveMonitors.removeAll()
        // Make sure the pointer isn't left hidden after unlock.
        NSCursor.setHiddenUntilMouseMoves(false)
    }

    private func startObservingScreenChanges() {
        stopObservingScreenChanges()
        screenObserver = NotificationCenter.default.addObserver(
            forName: NSApplication.didChangeScreenParametersNotification,
            object: nil,
            queue: .main
        ) { [weak self] _ in
            guard let self else { return }
            // Cancel any pending recreation — true debounce so only the last
            // notification in a burst triggers work.
            self.screenChangeWork?.cancel()
            let work = DispatchWorkItem { [weak self] in
                guard let self else { return }
                logger.info("Screen parameters changed — recreating overlay windows")
                for window in self.windows {
                    window.orderOut(nil)
                    window.contentView = nil
                    window.close()
                }
                self.windows.removeAll()
                if self.regionAllowedFrame != nil {
                    self.createRegionWindows()
                } else {
                    self.createWindows()
                }
            }
            self.screenChangeWork = work
            DispatchQueue.main.asyncAfter(deadline: .now() + 0.3, execute: work)
        }
    }

    private func stopObservingScreenChanges() {
        if let observer = screenObserver {
            NotificationCenter.default.removeObserver(observer)
            screenObserver = nil
        }
    }

    private func startObservingSessionChanges() {
        stopObservingSessionChanges()
        sessionObserver = NSWorkspace.shared.notificationCenter.addObserver(
            forName: NSWorkspace.sessionDidResignActiveNotification,
            object: nil,
            queue: .main
        ) { _ in
            NotificationCenter.default.post(name: .lockpawSessionLost, object: nil)
        }
    }

    private func stopObservingSessionChanges() {
        if let observer = sessionObserver {
            NSWorkspace.shared.notificationCenter.removeObserver(observer)
            sessionObserver = nil
        }
    }

    deinit {
        stopObservingScreenChanges()
        stopObservingSessionChanges()
        stopCursorConcealment()
        for window in windows {
            window.orderOut(nil)
            window.contentView = nil
            window.close()
        }
    }
}

private struct RegionMaskView: View {
    let onUnlock: () -> Void

    var body: some View {
        GeometryReader { proxy in
            ZStack {
                Color.black.opacity(0.72)

                if proxy.size.width >= 150 && proxy.size.height >= 90 {
                    Button {
                        DispatchQueue.main.async {
                            onUnlock()
                        }
                    } label: {
                        Label("Unlock", systemImage: "lock.open.fill")
                            .font(.system(size: 13, weight: .semibold))
                            .padding(.horizontal, 14)
                            .padding(.vertical, 8)
                            .background(.ultraThinMaterial, in: Capsule())
                            .overlay {
                                Capsule().stroke(.white.opacity(0.22), lineWidth: 1)
                            }
                    }
                    .buttonStyle(.plain)
                    .foregroundStyle(.white)
                }
            }
            .contentShape(Rectangle())
            .onTapGesture {
                DispatchQueue.main.async {
                    onUnlock()
                }
            }
        }
    }
}
