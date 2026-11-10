# PISKIN MUSIC — development plan

Use [CLAUDE.md](CLAUDE.md) for engineering instructions,
[ARCHITECTURE.md](ARCHITECTURE.md) for ownership and dependencies,
[DESIGN.md](DESIGN.md) for the proposed product, and [TESTING.md](TESTING.md)
for evidence and gates.

## Source audit — 2026-11-10

Upstream commit: [`8df198de9e3eccbb717d4370303cdb7f811a72a8`](https://github.com/k1y0miiii/applemusic-tui/commit/8df198de9e3eccbb717d4370303cdb7f811a72a8)
(2026-10-23, `fix(eq): draw the equalizer like the visualizer bars`).
Tree: `a3a494548ccef67a7b13801b3bf6284a6f8e4a34`.

The repeated provenance comparison used the commit-pinned GitHub recursive
tree: 79 blobs, with 78 unchanged files and one expected difference,
`.gitignore`. Its only change removes `DESIGN.md`; restoring that line in memory
reproduces the upstream blob hash. No upstream files are missing. The only
additional files are these five root documents, which were already present
when this review began. Environment-owned `.git`, `.agents`, `.codex`, and
`.aws` directories are excluded. Git blob SHA-1 values are computed as SHA-1
of `blob <byte length>\0<file bytes>`. [TESTING.md](TESTING.md#provenance-recheck)
records the API sources, manifest digest and reproduction procedure.

The initial restricted session could establish source contents but not local
branch history: `git status`, `git log`, and `git remote` failed because `.git`
was empty and read-only. After unrestricted access was enabled, the existing
history was cloned from the verified user-owned fork
[theonheptagrammaton/piskinmusic](https://github.com/theonheptagrammaton/piskinmusic)
and restored after comparing all 79 upstream files. The recovered `main` tracks
`origin/main` at the baseline commit; GitHub reports it unprotected with no
applicable branch rules. The user authorized commit and push to this fork.

MIT permits modification and redistribution with the copyright and permission
notice retained. LICENSE is unchanged. No dependency license audit beyond the
upstream project's own license has been completed; inspect dependency notices
before distributing binaries and any bundled browser components.

## Existing capabilities to preserve

| Capability | Inspected implementation |
| --- | --- |
| Browser login, session persistence, sign-out, reconnect/reload | `engine/engine.go`, `account.go`, root update loop |
| MusicKit playback, queue replacement/jump/play-next/append | `engine/musickit.go`, engine API, search overlay |
| Catalog songs/albums/playlists, library, recent items | MusicKit queries, queue/recent/search views |
| Pause/resume, seek, next/previous, volume, shuffle, repeat | engine transport and keyboard/mouse controls |
| Spectrum, torus, sphere; labeled simulated fallback | `visualizer/`, `orb.go`, root visualizer rendering |
| Linux system audio, macOS CoreAudio process tap | platform-specific visualizer sources |
| Lyrics, color artwork, recent cover grid | `lyrics/`, `artwork.go`, root views |
| EQ and saved presets/gains | Web Audio EQ in engine, `eq.go` |
| MPRIS, optional Last.fm scrobbling | `mpris/`, `desktop.go`, `lastfm/`, `scrobble.go` |
| Six themes, config, mouse, help, installers/releases | root support files, Makefile, GitHub workflows |

These are source capabilities, not confirmations of runtime behavior on this
machine. The current application is a dashboard with overlays, not four primary
screens. Its timeline waveform records band-derived loudness, not PCM samples.

## ARM64 and environment blockers

| Observation | Consequence / resolution |
| --- | --- |
| `uname -sm`: `Linux aarch64` | Correct architecture for target verification; full hardware/session behavior still needs testing. |
| `go` initially missing | RESOLVED for this run: official Go 1.26.11 ARM64 toolchain outside the checkout; tests and builds executed. |
| Chromium `153.0.8010.36 Arch Linux ARM` installed | Browser existence does not prove compatible Widevine, codecs, authorization, or full-track playback. |
| Audio access denied in the initial restricted session | RESOLVED with unrestricted desktop access: actual PipeWire Go source/service capture passed. |
| No usable Git metadata in the initial restricted session | RESOLVED: existing fork history recovered; `origin/main` verified as the authorized destination. |
| Widevine initially unverified | VERIFIED: full subscription track, attached media keys, audible output and real Go PCM on this host. No DRM changes. |

If official browser playback fails on Linux ARM64, stop UI expansion and record
the exact media/CDM error and supported recovery options. A macOS playback pass
is useful but does not satisfy the Linux ARM64 gate.

## Phases and acceptance gates

### 0 — Baseline and specification (complete; initial runtime limits recorded)

Inspect source, dependencies, license, tests, and workflows. Produce the five
root documents and record baseline evidence. Remove `DESIGN.md` from .gitignore
so the specification can be tracked. The ignore rule was already removed at
review entry and its exact upstream diff was verified. No application source
changes in this phase.

Validation: provenance comparison, document/link checks, diff review. Automated
Go tests were deferred to phase 1. The missing-metadata commit blocker is resolved. Suggested
commit: `docs: establish PISKIN MUSIC baseline and development gates`.

### 1 — Working playback and live capture (complete on target)

Run upstream tests first. Execute the Linux ARM64 subscription playback and
capture protocols in TESTING.md. Check headed login, special-workspace hiding,
pause/resume, track changes, reconnect, and real PCM during playback.

Add an opt-in Linux source/service smoke test under `visualizer/`, patterned
after the existing opt-in macOS smoke test. Record negotiated format, timestamps,
finite samples and signal level without retaining audio. Keep synthetic tests
explicitly separate. Correct only observed playback/capture failures, with
regressions around the affected boundary. Verify source naming against actual
server identity: Linux currently calls every PulseAudio server `PIPEWIRE`.

Exit: recorded real playback and Go capture passes on target hardware, baseline
test results, no unresolved playback blockers. Suggested commit:
`test: verify Linux ARM64 playback and system audio capture` (only after evidence).

Completed on 2026-11-10: audible subscription playback past two minutes on Linux
ARM64, real Go PCM/FFT capture correlated with pause/resume, transport/queue/EQ,
special-workspace placement and reload/reconnect. Baseline and final suites,
isolated server-loss/default-monitor tests and bounded headed login cancellation
are recorded in [TESTING.md](TESTING.md#target-verification--2026-11-10). Fixes
cover observed startup/disconnect races, canceled polling and truthful capture
status/history; no DRM or entitlement workaround was added. One catalog item
returned CONTENT_UNAVAILABLE and is explicitly excluded from playback passes.

### 2 — Separate application state and rendering (complete)

Incrementally extract root model responsibilities while preserving the current
layout and behavior. Introduce a narrow playback interface around existing engine
methods, injectable commands, explicit capture status, and a shared layout result
for rendering and hit testing. Start with root-package files (`app_state.go`,
`commands.go`, `layout.go`, `components.go`) to avoid an import-cycle-heavy rewrite;
these are proposed files, not existing directories. Keep `engine`, `visualizer`,
`mpris`, `lyrics`, and `lastfm` boundaries intact.

Exit: unchanged upstream regression suite plus command/failure/resize checks;
no extra browser polling or audio work. Commit the extraction independently.

Completed as a root-package extraction: playback interface, per-model command
factories, derived capture status and shared content/tile/progress geometry.
Existing regression tests remain unchanged. Command/failure/resize tests pass,
as do race, vet and cgo/nocgo builds. Sixty rendered frames match parent commit
`e816b69` byte for byte. Mouse hits now exclude hidden controls, borders, gaps
and padding. See the phase 2 evidence in TESTING.md. No live browser or audio
verification was repeated for this extraction.

### 3 — Monochrome shell and four primary screens (complete)

Extend centralized theme tokens and shared components. Add screen selection and
screen-specific layouts; retain search, queue, EQ, account and help access.
Extend `mouse.go`, `help.go`, and layout tests together. Preserve original-color
artwork with half-block fallback. No new image library required.

Intentional changes: PISKIN MUSIC branding, default monochrome shell, four-screen
navigation. Retain existing named palettes/configs as explicit legacy options;
the standard PISKIN appearance remains monochrome. Do not silently overwrite
saved configuration, authentication profiles, executable aliases or MPRIS IDs.

Exit: keyboard/mouse parity, narrow/wide/short layouts, all prior actions reachable,
no colored shell accents in the default theme. Commit the usable screen phase.

### 4 — PCM waveform and trustworthy visualizations

Extend `visualizer.Frame` with a bounded value-owned time-domain waveform and
sample metadata. Aggregate real PCM before discarding capture buffers. Keep the
existing spectrum analyzer and torus/sphere mathematics. Add waveform mode in
the UI; do not rename the existing loudness timeline as PCM. Display freshness
and source truthfully. Define demo as explicit opt-in; default capture failure
becomes a quiet unavailable panel, preserving demo as a documented compatibility
option. Existing tests expecting automatic simulated fallback need intentional
updates, not deletion.

Exit: deterministic PCM tests (silence, sine, impulse, stereo/anti-phase,
chunking), stale/error recovery checks, live target verification. Commit separately.

### 5 — Resource use, platform validation, release

Measure the current and new implementations on the same target. Cache static
artwork/layout output, bound waveform storage, suppress hidden-screen rendering,
and lower animation cadence when idle or reduced motion is requested. Keep
capture policy independent from screen visibility if the history needs it.

Exit: tests/vet/race/builds, macOS native capture checks, target performance
measurements, keyboard/mouse acceptance, dependency/license review, migration
notes and release packaging. Preserve Windows functionality and honest unsupported
capture status. Push completed commits only to a verified development branch.

## Current change scope

Phases 0–3 are complete. Phase 3 adds PISKIN MUSIC branding, a fixed monochrome
default, Library/Focus/Visualizer/Lyrics screens, responsive shared geometry,
queue/EQ/search/account/help access and keyboard/mouse navigation. Original
palette names and saved config/profile/alias/MPRIS identities remain compatible.
Original-color artwork uses the existing half-block renderer; no dependency was
added. See the phase 3 evidence in TESTING.md. PCM waveform, explicit demo
policy and resource/platform validation remain phases 4–5.
