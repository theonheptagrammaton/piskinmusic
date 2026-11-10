# PISKIN MUSIC — architecture

## Inspected baseline

Source commit: `8df198de9e3eccbb717d4370303cdb7f811a72a8` (see [PLAN.md](PLAN.md) for
provenance and runtime blockers). All source files are Go except platform audio
helpers, scripts, documentation, and workflow definitions. There is no frontend
bundle or browser-rendered application UI.

[DESIGN.md](DESIGN.md) specifies proposed interaction changes;
[TESTING.md](TESTING.md) records baseline evidence and verification gates.

```text
Keyboard / mouse ──> Bubble Tea model (root main package)
                              │ asynchronous commands / state messages
                              v
                    engine.Engine ──CDP──> headed Chromium
                              ^              music.apple.com / MusicKit
                              │              official sign-in + DRM playback
                         state polling                │ audio output
                              │                       v
                         MPRIS state            system audio mixer
                                                      │
                       UI frame <── visualizer.Service <── Source
                                     Analyzer / FFT     Linux monitor or macOS tap

Root commands also fetch artwork, LRCLIB lyrics and opt-in Last.fm data.
```

### Existing ownership

| Area | Files/packages and behavior |
| --- | --- |
| App state, commands, layout, views | `main.go` wires startup; `app_state.go` owns the model, messages and Init/Update; `commands.go` wraps effects and timers; `layout.go` owns shared geometry; `components.go` renders the dashboard and its components. |
| Support views and input | `mouse.go`, `help.go`, `overlay.go`, `account.go`, `eq.go`, `orb.go`. |
| Theme/config/art | `theme.go` holds palettes, mutable global tokens, simple TOML-subset parsing and persistence; `artwork.go` downloads/caches and renders images as colored half blocks. |
| Playback | `engine.Engine` serializes CDP access with a mutex; `State` exposes transport/queue/auth/EQ/error snapshots; JS in `engine/musickit.go` uses the page's MusicKit instance and playback recovery logic. |
| Browser lifecycle | Headed browser with persistent profile; visible official sign-in if needed, then platform parking/minimization. Hyprland uses PID-targeted special-workspace moves with placement checks and legacy dispatch fallback. |
| Audio | `visualizer.Source` provides format, name, cancellable float32 reads, close. Linux records the default sink monitor with pure-Go PulseAudio; macOS uses a CoreAudio process tap through Objective-C/cgo. |
| Analysis | 4096-frame Hann FFT, 32 frequency bands, stereo power handling, attack/release smoothing. The service owns its goroutine, analyzer, cancellation and latest atomic snapshot. |
| Integration | `mpris` depends on a small Controls interface, not the UI; `desktop.go` adapts the engine. `lyrics` calls LRCLIB; `lastfm` handles optional authentication/scrobbling, with listening-time accounting in `scrobble.go`. |

The UI ticks at 30 Hz and polls playback separately (500 ms). Frames currently
contain bands, source, Live, and timestamp; no time-domain waveform. The UI
checks frame freshness at 400 ms. Target verification corrected the capture
boundary: opening waits for the first fresh frame, stale capture loses LIVE,
and non-live data cannot feed timeline history. Fresh zero PCM remains live.
The labeled simulated fallback remains available; it does not record audio history.

The Linux source is system-output capture, not guaranteed Apple-Music-only audio:
other applications on the selected sink can affect it. It selects the default
sink when opened; reopening selects the current monitor. Isolated default-sink
and monitor-removal checks passed (see TESTING.md); automatic following of
default-device changes and physical hotplug are not claimed.

## Dependencies

Declared direct modules from [go.mod](go.mod) (Go 1.26):

| Module | Version | Role |
| --- | --- | --- |
| charmbracelet/bubbletea | v1.3.10 | Event loop and terminal input/rendering |
| charmbracelet/lipgloss | v1.1.0 | Terminal layout/style |
| charmbracelet/x/ansi | v0.10.1 | ANSI-aware text handling |
| chromedp/cdproto | v0.0.0-20260714215040-dc233986426f | Chromium protocol types |
| chromedp/chromedp | v0.16.0 | Browser control |
| godbus/dbus/v5 | v5.2.2 | MPRIS / session bus |
| jfreymuth/pulse | v0.1.2 | Linux monitor capture |
| lucasb-eyer/go-colorful | v1.2.0 | Artwork color/theme processing |
| muesli/termenv | v0.16.0 | Terminal capabilities |
| gonum.org/v1/gonum | v0.17.0 | FFT |

There are 10 direct and 19 indirect requirements. Indirect versions are pinned
in [go.mod](go.mod), with checksums in [go.sum](go.sum): terminal/Unicode helpers,
chromedp sysutil, experimental JSON, websocket/http helpers, and Go x/sys/x/text.
Keep the lock data and avoid adding dependencies for the proposed screen shell.

Runtime requirements: a compatible headed Chromium with the official DRM/media
stack, an Apple Music subscription and network access; pipewire-pulse or
PulseAudio on Linux; session D-Bus for MPRIS; macOS 14.2+ and cgo for its live
tap. Linux can build without cgo. A darwin build without cgo lacks live capture.

## Root extraction — phase 2

The root `playback` interface lists only engine operations consumed by root
commands and desktop controls. Production still uses `*engine.Engine`, with its
existing snapshot types and lifecycle. A per-model `commandSet` supplies optional
factories for connection, capture opening, metadata and timers, plus a hook for
MPRIS publication; nil fields retain production defaults. Playback commands capture their engine and
request arguments before asynchronous execution. Reconnect closes the old engine
before executing the replacement connection command.

`captureStatus()` derives starting, live, stale, unavailable or demo from the
existing service lifecycle flags, avoiding a second stored source of truth.
Rendering and history eligibility use that status. The existing SIMULATED label
and automatic simulated fallback after unavailable capture are preserved for this
phase; the explicit-demo product change remains phase 4 work.

The shared layout contains panel sizes, visible content rectangles, recent tile
rectangles and progress geometry. View and mouse input consume the same result;
recent cover sizing has one implementation. Hidden controls, borders, tile gaps
and blank padding do not activate playback. Frame rendering remains unchanged.
There are no new packages, dependencies, browser polls or audio reads.

## Intended boundaries — incremental changes

1. **Playback:** keep `engine` responsible for browser lifecycle and MusicKit.
   Introduce a root-facing interface for the methods actually consumed, retaining
   engine data types initially. Do not move browser code into render components.
2. **Application:** root application state owns selected screen, focused region,
   queue selection, requests and transient messages. Commands perform effects;
   messages reduce them into state. Preserve track/request identity checks so
   late responses cannot overwrite a newer track. MPRIS reads the same playback
   snapshot used by the TUI.
3. **Audio:** `Source -> Analyzer -> Service -> Frame` remains independent of
   playback and terminal rendering. Extend snapshots with bounded PCM-derived
   waveform values and format metadata; do not expose reusable capture slices
   across goroutines. Keep cancellation and idempotent cleanup.
4. **Rendering:** extract pure screen/component functions consuming snapshots,
   semantic theme tokens and shared layout rectangles. Mouse hit testing must
   use those exact rectangles, including clipping/hidden regions.

Capture state should explicitly distinguish starting, live, stale, unavailable,
and demo. Fresh silence is valid live capture. Staleness uses the last captured
frame time rather than playback progress; terminal source errors invalidate
LIVE immediately. Demo data is generated only through an explicit mode, carries
demo identity, and never contaminates recorded audio history.

Use existing root-package files first; introduce internal packages only when a
real dependency boundary benefits. Avoid a wholesale directory migration.

## Persistence, compatibility and resource policy

Keep current config/profile locations during initial work. A source discrepancy
needs attention before any migration: root `configDir()` honors
`AMTUI_CONFIG_DIR`, but engine `profileDir()` hardcodes `~/.config/amtui/chrome`.
The override therefore does not isolate login profiles. Fix/test that separately
before relying on isolated integration profiles.

Preserve MPRIS service identity `org.mpris.MediaPlayer2.amtui`, aliases, and
existing persisted theme/EQ/viz state until migration behavior is documented.
Do not rename IDs just to rebrand display text.

Reuse latest-frame publication instead of queueing redraws. Keep memory bounded,
cache artwork by identity/size, avoid redecoding per tick, and skip inactive
screen rendering. Measure Chrome and Go resource use separately. No renderer
may stall audio reads or the UI on browser/network operations.
