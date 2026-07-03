<h1 align="center">Region Lockpaw</h1>

<p align="center">
  <strong>Drag a usable rectangle on macOS. The selected area stays interactive; the rest of the screen is masked until authentication.</strong>
</p>

<p align="center">
  <img src="https://img.shields.io/badge/macOS%2014+-111?style=flat-square&logo=apple&logoColor=fff" alt="macOS 14+">
  <img src="https://img.shields.io/badge/Swift%205.9-111?style=flat-square&logo=swift&logoColor=F05138" alt="Swift 5.9">
  <img src="https://img.shields.io/badge/MIT-111?style=flat-square" alt="MIT License">
</p>

---

This is a public fork-based development project built from [Lockpaw](https://github.com/sorkila/lockpaw). The original Lockpaw full-screen lock remains available; this fork adds a region-lock mode.

## Features

- **Region lock** - choose `Lock Region...` from the menu bar, then drag a rectangle.
- **Mouse works inside the rectangle** - no overlay window is placed over the selected area.
- **Keyboard works inside the active app** - region lock does not start Lockpaw's keyboard-blocking event tap, so normal keyboard input continues to the foreground application.
- **Frame outside is masked** - the rest of the selected screen is covered with four mask windows; other displays are fully masked.
- **Authentication to leave the mask** - clicking the masked area triggers the existing Touch ID / macOS password unlock flow.
- **Original Lockpaw screen lock remains** - `Lock Screen` still uses the upstream full-screen privacy lock behavior.

<br>

## Usage

| Action | How |
|--------|-----|
| Region lock | Menu bar icon -> `Lock Region...`, then drag the usable rectangle |
| Full-screen lock | Menu bar icon -> `Lock Screen`, or the configured hotkey |
| Unlock region mask | Click the masked area and authenticate |
| Unlock full-screen lock | Use the hotkey, Touch ID button, or password button |
| Settings | Menu bar -> `Settings...` |

<br>

## Install

### Download

Grab the latest signed & notarized DMG from [getlockpaw.com](https://getlockpaw.com) or [GitHub Releases](https://github.com/sorkila/lockpaw/releases).

### Homebrew

```bash
brew tap sorkila/lockpaw
brew install --cask lockpaw
```

### Build from source

```bash
brew install xcodegen
git clone https://github.com/william08190/region-lockpaw.git
cd region-lockpaw
xcodegen generate
xcodebuild -scheme Lockpaw -configuration Release build
```

On first launch, grant **Accessibility** when prompted. The Lockpaw icon appears in your menu bar.

<br>

## Design

The lock screen is intentionally minimal. Near-black canvas. Subtle radial glow. One element at a time.

**Calm by default** — the screen opens with your chosen mascot, your message, and a quiet elapsed timer; the pointer slips away after a moment of stillness. The fallback auth button waits quietly at the bottom — always there, never loud. When an agent pings, the screen breathes two slow waves of teal, then keeps a soft "your agent needs you" hint until you return.

**Mascots** — a metallic origami dog or cat rendered in teal and amber, floating in a pool of light. Slow 12-second breathing cycle. On successful unlock, the mascot scales up with a teal bloom and fades away.

**Typography** — system San Francisco throughout. Regular weight message at 55% white. Monospaced timer at 35%. The screen whispers.

**Auth button** — glass material effect with a subtle border. Visible enough to be tappable, quiet enough to stay out of the way.

<br>

## Under the hood

**Hotkey** — `CGEvent.tapCreate` with `.listenOnly` on a dedicated background thread. Bypasses the LSUIElement activation issue that affects Carbon hotkeys in menu bar apps. Requires Accessibility permission.

**Input blocking** — separate `CGEventTap` intercepts all keyboard, scroll, and tablet events system-wide while locked. Mouse events pass through to the overlay (SwiftUI buttons need clicks). If macOS disables the tap, it re-enables synchronously in the callback.

**Window level** — `CGShieldingWindowLevel()`, the highest level in the system. Above Spotlight, Notification Center, screen savers, everything.

**Multi-display** — one overlay window per screen, recreated on hot-plug.

**State machine** — `LockState` enum with validated transitions. Every `transitionTo()` call is checked. State is verified again after async authentication returns.

**Sleep prevention** — `IOPMAssertion` keeps the Mac awake while locked.

**Auth** — `LAContext.evaluatePolicy(.deviceOwnerAuthentication)` for Touch ID with password fallback. Rate-limited: 30s cooldown after 3 failed attempts.

**Auto-updates** — Sparkle framework checks for updates automatically. Appcast hosted at getlockpaw.com.

<br>

## Security model

Lockpaw is a **visual privacy tool**, not a security boundary.

It guards against the accidental — a colleague, a cat, your own muscle memory while agents run. Not the intentional.

<details>
<summary><strong>What it does</strong></summary>
<br>

- Overlay at highest system window level
- Event tap blocks all keyboard/scroll input
- Fast User Switching cancels auth, keeps lock active
- Accessibility revocation detected and handled (force unlock with warning)
- URL scheme rate-limited (100ms debounce)
- Debug escape hatch compile-gated (`#if DEBUG`)
- State machine validates every transition
- Hotkey conflict detection against system shortcuts

</details>

<details>
<summary><strong>What it doesn't do</strong></summary>
<br>

- Prevent `pkill Lockpaw`
- Block synthetic events (AppleScript, Accessibility API)
- Survive kernel-level access
- Protect against screen recording during overlay fade-in

For real security: `Ctrl+Cmd+Q`.

</details>

Found a lock or auth bypass anyway? Please [report it privately](SECURITY.md).

<br>

## URL scheme

```
lockpaw://lock              Lock the screen
lockpaw://unlock            Unlock with Touch ID
lockpaw://unlock-password   Unlock with password
lockpaw://toggle            Toggle lock state
```

<br>

## Architecture

```
Lockpaw/
├─ LockpawApp                     Entry, MenuBarExtra, AppDelegate, onboarding
├─ Controllers/
│  ├─ LockController              State machine, lock/unlock orchestration
│  ├─ Authenticator               LAContext · Touch ID · password fallback
│  ├─ InputBlocker                CGEventTap · keyboard/scroll blocking
│  ├─ HotkeyManager               CGEventTap · global hotkey detection
│  ├─ OverlayWindowManager        NSWindow · multi-display · shielding level
│  ├─ SleepPreventer              IOKit · idle sleep assertion
│  └─ AgentNotifier               UNUserNotificationCenter · agent-ping notifications
├─ Models/
│  ├─ LockState                  .unlocked → .locking → .locked → .unlocking
│  ├─ HotkeyConfig               Centralized hotkey UserDefaults access
│  ├─ PingDecision               Pure agent-ping decision (pulse/notify/sound)
│  └─ Mascot                     Dog/cat lock screen preference
├─ Views/
│  ├─ LockScreenView             Dog/cat mascot · agent-ping glow · fallback auth
│  ├─ AmbientScreenView          Secondary display gradient animation
│  ├─ MenuBarView                Dropdown · lock/unlock/quit
│  ├─ SettingsView               Native tabs · hotkey recorder · updates
│  └─ OnboardingView             5-step wizard · hotkey · accessibility · agent alerts
├─ Utilities/
│  ├─ Constants                  Timing, animations, formatting
│  ├─ Notifications              All Notification.Name in one place
│  └─ AccessibilityChecker       AXIsProcessTrusted + System Settings
└─ Resources/
   └─ Assets                      App icon, mascot, menu bar icon, colors

LockpawCLI/
└─ main                           `lockpaw` CLI · ping · install-cli · install-hook
```

<br>

## CI

Pushes to `main` and PRs run build + 50 unit tests via GitHub Actions. Shipped DMGs are Developer ID-signed, notarized, and published to [GitHub Releases](https://github.com/sorkila/lockpaw/releases); auto-updates are delivered through Sparkle with EdDSA-signed appcasts.

<br>

---

<p align="center">
  <sub>
    <a href="https://getlockpaw.com">getlockpaw.com</a>
  </sub>
</p>

<br>
