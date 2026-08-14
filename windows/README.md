# Region Lockpaw for Windows

Region Lockpaw is a Windows 10/11 notification-area utility that keeps one
dragged screen region usable while masking the rest of every connected display.

## Install

Run `RegionLockpawSetup-1.0.0-x64.exe`. The per-user installer does not request
administrator access. It installs under `%LOCALAPPDATA%\Programs\Region Lockpaw`,
adds Start menu and uninstall entries, and enables start-at-sign-in by default.

The current installer is not code-signed, so Microsoft Defender SmartScreen may
show an unrecognized-app warning. Verify the SHA-256 checksum before running it.
The build writes checksums to `dist/SHA256SUMS.txt`.

## Use

- Open the notification-area menu and choose **Lock Region**, or press
  `Ctrl+Alt+L`.
- Drag the usable rectangle. Dragging right-to-left or across negative-coordinate
  monitors is supported; a region stays within the display where the drag began.
- The selected rectangle remains mouse- and keyboard-accessible. Other displays
  are completely masked.
- Press `Ctrl+Alt+L` again for quick unlock.
- Click a masked area to unlock through the Windows secure sign-in screen. The
  mask is removed only after the Windows session is unlocked.
- Locking the workstation independently with `Win+L` never removes an existing
  region mask; it is restored when you return to the session.
- Press `Esc` or right-click while selecting to cancel.

The tray menu can toggle start-at-sign-in and uninstall is available from Windows
Settings or the Start menu.

## Security Model

Region Lockpaw is a visual privacy and accidental-input guard, not a Windows
security boundary. It cannot cover the secure desktop, UAC prompts,
`Ctrl+Alt+Delete`, or protect against Task Manager/process termination. Use
`Win+L` when the entire Windows session must be secured.

The app never receives or stores a Windows password. Secure unlock delegates to
`LockWorkStation` and accepts the operating system's session-unlock event only
when Region Lockpaw initiated that specific secure-sign-in flow.

## Build

Windows prerequisites:

- Go 1.25 or later
- NSIS 3.x (`choco install nsis`)

```powershell
cd windows
./scripts/build.ps1 -Version 1.0.0
```

macOS cross-build prerequisites:

- Go 1.25 or later
- NSIS 3.x (`brew install makensis`)

```bash
cd windows
./scripts/build.sh 1.0.0
```

Artifacts are written to `build/windows-amd64/RegionLockpaw.exe` and
`dist/RegionLockpawSetup-1.0.0-x64.exe`.

Runtime logs are capped and stored under `%LOCALAPPDATA%\Region Lockpaw`.
