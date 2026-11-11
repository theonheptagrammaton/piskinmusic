# PISKIN MUSIC — architecture

## Unified dashboard — current presentation

`dashboard.go` composes Library, Queue, Cover, Visualizer and Lyrics from the
rectangles owned by `layout.go`. All five retain content at 40×12 and above.
Three columns are used from 100 columns, otherwise the three listening panels
stack beside the two lists. The optional recent cover grid remains available
in tall, wide windows; search, expanded queue, EQ, account and help own their
covered region. Mouse hit testing consumes the same list/tile/transport geometry.

The existing screen enum now identifies a highlighted dashboard panel rather
than hiding other primary content. Visible visualization therefore ticks on
all panel highlights, with the existing idle/overlay/blur/reduced-motion policy.
`theme.go` owns the PISKIN spectrum used by headings, selections, spectrum bars,
shape painting and PCM traces. Artwork and the audio/capture/playback paths are
unchanged. Historical phase descriptions below describe the earlier layout.

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
| App state, commands, layout, views | `main.go` wires startup; `app_state.go` owns the model, messages and Init/Update; `commands.go` wraps effects and timers; `layout.go` owns shared geometry; `components.go` renders shared playback components and dispatches screens; `screens.go` renders primary screen content and owns screen/focus transitions. |
| Support views and input | `mouse.go`, `help.go`, `overlay.go`, `account.go`, `eq.go`, `orb.go`. |
| Theme/config/art | `theme.go` holds palettes, mutable global tokens, simple TOML-subset parsing and persistence; `artwork.go` downloads/caches and renders images as colored half blocks. |
| Playback | `engine.Engine` serializes CDP access with a mutex; `State` exposes transport/queue/auth/EQ/error snapshots; JS in `engine/musickit.go` uses the page's MusicKit instance and playback recovery logic. |
| Browser lifecycle | Headed browser with persistent profile; visible official sign-in if needed, then platform parking/minimization. Hyprland uses PID-targeted special-workspace moves with placement checks and legacy dispatch fallback. |
| Audio | `visualizer.Source` provides format, name, cancellable float32 reads, close. Linux records the default sink monitor with pure-Go PulseAudio; macOS uses a CoreAudio process tap through Objective-C/cgo. |
| Analysis | 4096-frame Hann FFT, 32 frequency bands, stereo power handling, attack/release smoothing. The service owns its goroutine, analyzer, cancellation and latest atomic snapshot. |
| Integration | `mpris` depends on a small Controls interface, not the UI; `desktop.go` adapts the engine. `lyrics` calls LRCLIB; `lastfm` handles optional authentication/scrobbling, with listening-time accounting in `scrobble.go`. |

The UI ticks at up to 30 Hz for visible playing visualization, 10 Hz with
`ui.reduced_motion`, and 4 Hz on other/idle/hidden screens. Boot/loading uses
10 Hz unless motion is reduced. Playback polling remains separate (500 ms). Frames contain
bands, a fixed-size PCM envelope, negotiated format, total complete frames read,
source, Live, and local read-completion timestamp. The UI checks freshness at
400 ms: opening waits for the first fresh PCM frame, stale capture clears its
traces, and fresh silence remains live. Terminal errors invalidate capture
immediately and show a quiet unavailable panel. Only explicit
`visualizer.demo = true` enables the legacy simulated bands when capture is
unavailable; demo never supplies PCM or timeline history.

The Linux source is system-output capture, not guaranteed Apple-Music-only audio:
other applications on the selected sink can affect it. It selects the default
sink when opened; reopening selects the current monitor. Isolated default-sink
and monitor-removal checks passed (see TESTING.md); automatic following of
default-device changes and physical hotplug are not claimed.

## PCM snapshots — phase 4

`Service.run` validates PCM through the unchanged Analyzer, then accumulates it
in its own fixed 4096-frame stereo ring before reusing the read buffer. Reads
that split stereo frames retain at most one sample for the next read. The FFT
still receives whole frames, with its original Hann window, band mapping,
stereo power handling and attack/release math. Torus/sphere geometry is unchanged.

Every published `Frame` owns a `Waveform` value: up to 128 chronological min/max
points per channel, per-channel RMS/absolute peak, and the number of actual PCM
frames covered. Startup windows contain only received samples. For point i,
the frame range is `[i*Frames/Count, (i+1)*Frames/Count)`; the complete window is
the latest 4096 received frames, about 85.3 ms at 48 kHz. Channels stay separate
so anti-phase audio cannot disappear through downmixing. Signed source amplitude
is retained; only display coordinates clip at full scale. There are no shared
slices in the envelope, and `Latest` returns a value copy. `TotalFrames` counts
received complete frames, not dropped source chunks or elapsed hardware time.

The persisted `waveform` mode renders this envelope with separate L/R traces
and system-output/format/window metadata. Narrow columns merge extrema rather
than dropping impulses. Existing shape IDs, mode names and the `orb` alias remain
compatible. The transport's `wave` structure remains a band-derived loudness
timeline along the track; it is not PCM. Starting/stale/unavailable states clear
PCM and peaks, while fresh frames restore the current source and metadata. View does no capture,
analysis, playback calls or wall-clock reads. Freshness uses receipt time, not a
hardware sample-clock timestamp; a backend label does not attribute sound to a
particular player.

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

## Monochrome screens — phase 3

The root model retains a selected primary screen, per-screen focus, library
snapshot/category selections and queue overlay state. Library snapshots remain
separate from catalog search results. Screen selection schedules no browser or
audio work. Rendering dispatches only the active primary content. The queue is
available as a full main-area overlay everywhere and as a support column when
space permits; EQ uses the main area while search/account/help take precedence.

`layout.go` owns navigation, toolbar, list, cover and progress rectangles. Search,
help, account and EQ each share their action geometry with pointer handling.
Mouse commands reuse keyboard command paths. ANSI/Unicode clipping bounds all
frames to the terminal, while lists scroll their selected row into view. Help
paginates in narrow/short terminals and exposes general commands to pointer
users. Screen shortcuts never consume search text or pass through blocking layers.

Theme tokens, including background and inverse selection text, live in theme.go.
The new `piskin` default is fixed monochrome; original named palettes and their
color overrides remain explicit legacy choices. Color-only existing configs
without a named/saved palette retain their original apple base. Resolving a theme writes no
configuration. Only an explicit theme/EQ/viz action persists that preference.
Authentication paths, module/executable aliases, engine APIs, MPRIS IDs, audio
service and visualization IDs are unchanged. The existing half-block artwork
renderer retains source colors, including inside the black shell. No terminal
image protocol or new image dependency was added.

## Resource policy — phase 5

The UI event loop owns a shared per-model cache across Bubble Tea value copies.
It retains at most 16 rendered artwork variants and 32,768 cells in total,
keyed by immutable image identity, dimensions and terminal color profile.
Oversized/non-comparable images use the original renderer without retention.
Replacement images and resizes cannot reuse stale cells. Decoded track covers
retain the existing eight-cover cache. A single static geometry entry is keyed
by size, screen, queue/EQ state and recent-content presence. Clock/volume/
shuffle/repeat transport geometry stays dynamic; rendering and hit testing
read the same result. The header cache also tracks palette and color profile.

Only visible visualization advances shape geometry/peaks; reduced motion freezes
orientation and removes the kick/pulse while real data can still update. Blur
reporting freezes the last terminal frame until focus returns. Screen/overlay
dispatch avoids rendering covered primary content. Capture/analysis and history
eligibility remain independent of visibility; each tick consumes only the latest
owned frame, without queueing missed UI frames. Adaptive band smoothing uses
elapsed time. Notices and initialization timeouts use elapsed timer timestamps;
optimistic progress and scrobbling cap suspend gaps at one second of listening.
All waveform storage remains fixed (4096 PCM frames, 128 envelope points,
240 track loudness buckets); no recording grows with track duration.
