# PISKIN MUSIC — engineering instructions

## Product and priorities

Build a premium, keyboard-first Apple Music terminal client for Linux and macOS.
The primary target is an M1 MacBook Pro running Linux ARM64, Omarchy,
Hyprland 0.56.2, Ghostty, and PipeWire. Use a black-and-white interface with
original-color album artwork. The four primary screens are Library, Focus,
Visualizer, and Lyrics. Retain mouse interaction and all existing functionality.

Working, authorized playback comes before visual development. Read [PLAN.md](PLAN.md),
[ARCHITECTURE.md](ARCHITECTURE.md), [DESIGN.md](DESIGN.md), and
[TESTING.md](TESTING.md) before implementation.

## Baseline

- Upstream: https://github.com/k1y0miiii/applemusic-tui
- Exact source baseline: `8df198de9e3eccbb717d4370303cdb7f811a72a8`.
- License: MIT, copyright (c) 2026 k1y0miiii. Preserve [LICENSE](LICENSE) and attribution
  in source and distributed artifacts.
- This is a Go 1.26 application, not a web UI. Keep the existing module path
  until an intentional module migration is needed.
- Source is organized around a root `main` package and `engine`, `visualizer`,
  `mpris`, `lyrics`, and `lastfm` packages. Do not assume a different layout.

## Hard gates

1. Establish the upstream baseline, run existing tests, and record failures.
2. Verify real Apple Music subscription playback on Linux ARM64 through the
   existing headed Chromium/MusicKit engine before major UI work.
3. Verify actual PCM capture from PipeWire through pipewire-pulse using the
   application's Go source, not merely the presence of audio command-line tools.
4. Record evidence and pass/fail status in TESTING.md. Compiling, mocked JS,
   synthetic PCM tests, changing playback position, and a LIVE label alone do
   not prove audible subscription playback or successful capture.

The initial session could not pass these gates: Go is missing, audio sockets
are inaccessible, and Git metadata is unavailable. Do not mark them passed.

## Boundaries

- Keep official browser sign-in, MusicKit, Chromium/CDP, and DRM intact.
  Never request Apple ID passwords in the terminal or extract protected streams.
- Keep authentication cookies, tokens, browser profiles, and personal account
  information out of repository files, test fixtures, logs, and screenshots.
- Preserve the playback lifecycle, initialization watchdog, and reconnect logic
  unless a separately tested fix requires changing them.
- Capture and FFT work stay outside rendering. UI consumes bounded immutable
  snapshots; no audio reads, network calls, or browser calls in View.
- Clearly distinguish live, starting, stale, unavailable, and demo data.
  Never present generated motion as live audio. Do not derive a PCM waveform
  from FFT magnitude bands or a playback timeline.
- Reuse Bubble Tea, Lip Gloss, the audio service, and existing dependencies.
  Justify additions by a concrete unmet need.
- Keep theme decisions centralized and keyboard/mouse geometry consistent.
- No desktop configuration changes or installer execution as part of UI work.

## Delivery workflow

Implement one independently testable phase at a time. Run the relevant checks
and inspect the diff after each phase. Document intentional behavior changes
and preserve upstream compatibility by default. Commit each completed phase
with a descriptive message after checks pass (or explicitly report blockers).

Before committing or pushing, inspect the real checkout's branch and remotes.
Push only to its configured development branch, never upstream or a protected
branch. Do not invent a remote, infer a branch from the directory name, or
replace missing Git history with a new repository. The initial restricted
session exposed an empty read-only `.git` directory. After access was restored,
the existing history was recovered from the verified user-owned fork
`theonheptagrammaton/piskinmusic`; `main` tracks `origin/main` and was confirmed
unprotected with no applicable branch rules. Go and runtime gates remain open.
