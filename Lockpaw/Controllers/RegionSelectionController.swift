import AppKit
import os.log

private let logger = Logger(subsystem: "com.eriknielsen.lockpaw", category: "RegionSelection")

@MainActor
final class RegionSelectionController {
    private var windows: [RegionSelectionWindow] = []
    private var completion: ((NSRect?) -> Void)?

    func begin(completion: @escaping (NSRect?) -> Void) -> Bool {
        cancel(notify: false)
        self.completion = completion

        let screens = NSScreen.screens
        guard !screens.isEmpty else {
            logger.error("Region selection failed - no screens available")
            self.completion = nil
            return false
        }

        for screen in screens {
            let window = RegionSelectionWindow(screen: screen) { [weak self] rect in
                self?.finish(with: rect)
            }
            windows.append(window)
            window.orderFrontRegardless()
        }

        NSApp.activate(ignoringOtherApps: true)
        windows.first?.makeKey()
        return true
    }

    func cancel(notify: Bool = true) {
        closeWindows()
        guard notify, let completion else {
            self.completion = nil
            return
        }
        self.completion = nil
        completion(nil)
    }

    private func finish(with rect: NSRect?) {
        closeWindows()
        guard let completion else { return }
        self.completion = nil
        completion(rect)
    }

    private func closeWindows() {
        for window in windows {
            window.orderOut(nil)
            window.contentView = nil
            window.close()
        }
        windows.removeAll()
    }
}

private final class RegionSelectionWindow: NSWindow {
    private let onComplete: (NSRect?) -> Void

    init(screen: NSScreen, onComplete: @escaping (NSRect?) -> Void) {
        self.onComplete = onComplete
        super.init(
            contentRect: screen.frame,
            styleMask: .borderless,
            backing: .buffered,
            defer: false,
            screen: screen
        )

        setFrame(screen.frame, display: true)
        level = NSWindow.Level(rawValue: Int(CGShieldingWindowLevel()) + 1)
        collectionBehavior = [.canJoinAllSpaces, .fullScreenAuxiliary, .stationary]
        isOpaque = false
        backgroundColor = .clear
        ignoresMouseEvents = false
        acceptsMouseMovedEvents = true
        hasShadow = false
        isReleasedWhenClosed = false
        contentView = RegionSelectionView(frame: NSRect(origin: .zero, size: screen.frame.size)) { [weak self] localRect in
            guard let self else { return }
            guard let localRect else {
                self.onComplete(nil)
                return
            }
            let global = NSRect(
                x: self.frame.minX + localRect.minX,
                y: self.frame.minY + localRect.minY,
                width: localRect.width,
                height: localRect.height
            )
            self.onComplete(global)
        }
    }

    override var canBecomeKey: Bool { true }
    override var canBecomeMain: Bool { true }

    override func keyDown(with event: NSEvent) {
        if event.keyCode == 53 {
            onComplete(nil)
        } else {
            super.keyDown(with: event)
        }
    }
}

private final class RegionSelectionView: NSView {
    private let minimumSize: CGFloat = 48
    private let onComplete: (NSRect?) -> Void
    private var dragStart: NSPoint?
    private var dragCurrent: NSPoint?

    init(frame frameRect: NSRect, onComplete: @escaping (NSRect?) -> Void) {
        self.onComplete = onComplete
        super.init(frame: frameRect)
        wantsLayer = true
        layer?.backgroundColor = NSColor.clear.cgColor
    }

    required init?(coder: NSCoder) {
        nil
    }

    override var acceptsFirstResponder: Bool { true }

    override func mouseDown(with event: NSEvent) {
        dragStart = convert(event.locationInWindow, from: nil)
        dragCurrent = dragStart
        needsDisplay = true
    }

    override func mouseDragged(with event: NSEvent) {
        dragCurrent = convert(event.locationInWindow, from: nil)
        needsDisplay = true
    }

    override func mouseUp(with event: NSEvent) {
        dragCurrent = convert(event.locationInWindow, from: nil)
        guard let rect = currentRect, rect.width >= minimumSize, rect.height >= minimumSize else {
            onComplete(nil)
            return
        }
        onComplete(rect)
    }

    override func draw(_ dirtyRect: NSRect) {
        NSColor.black.withAlphaComponent(0.55).setFill()
        bounds.fill()

        guard let rect = currentRect else { return }

        if let context = NSGraphicsContext.current?.cgContext {
            context.saveGState()
            context.setBlendMode(.clear)
            context.fill(rect)
            context.restoreGState()
        }

        let path = NSBezierPath(roundedRect: rect, xRadius: 8, yRadius: 8)
        NSColor.systemTeal.setStroke()
        path.lineWidth = 3
        path.stroke()

        let fill = NSBezierPath(roundedRect: rect.insetBy(dx: 1.5, dy: 1.5), xRadius: 7, yRadius: 7)
        NSColor.systemTeal.withAlphaComponent(0.08).setFill()
        fill.fill()
    }

    private var currentRect: NSRect? {
        guard let start = dragStart, let current = dragCurrent else { return nil }
        return NSRect(
            x: min(start.x, current.x),
            y: min(start.y, current.y),
            width: abs(current.x - start.x),
            height: abs(current.y - start.y)
        )
    }
}
