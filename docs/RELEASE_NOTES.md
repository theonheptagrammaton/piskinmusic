# Resource optimization development candidate

This candidate builds on `1e2b3be` (the PCM waveform phase). It adds bounded
artwork cell caching, cached screen/header geometry, hidden-screen suppression,
and adaptive UI cadence. See [TESTING.md](../TESTING.md) for measured results
and the remaining hardware release gates. Development archives are not a tagged
release or a claim that every runtime gate has passed.

## Migration

No files need moving. Executable aliases (`amtui`, `applemusic-tui`, `applemusic`),
the Go module path, browser profile, MPRIS identity, theme/EQ/visualizer mode IDs,
and existing configuration locations remain compatible. No dependencies changed.
The browser continues to handle official Apple Music sign-in and DRM.

An optional setting in `config.toml` is read at startup:

```toml
[ui]
reduced_motion = true
```

The default is false. Reduced motion holds torus/sphere orientation still,
disables bass pulse, and limits visible playing visualization updates to 10 Hz.
Normal visible playing visualization targets at most 30 Hz. Other screens,
paused playback and a terminal that reports blur use 4 Hz. Boot/loading uses
10 Hz unless reduced motion is enabled. A pending slow tick can take up to
250 ms to adopt a faster cadence. Keyboard/mouse messages still run immediately.

Primary screens covered by overlays are not rendered. Terminals supporting
focus reporting keep their last frame while blurred and refresh on focus.
Capture, analysis, playback polling and live loudness history continue while
the screen is hidden. History is sampled at the current UI cadence; it is a
bounded loudness summary, not a lossless PCM recording. The 4096-frame PCM ring,
128-point stereo envelope and 240-bucket track timeline retain their existing
fixed bounds. No audio files are saved.

The previous phase intentionally disabled automatic simulated capture fallback.
Windows and macOS builds without cgo show UNAVAILABLE for system capture;
Windows playback and input remain supported. Existing bars/torus/sphere IDs and
the waveform ID are unchanged. `[visualizer] demo = true` explicitly opts into
labeled DEMO/SIMULATED bands when capture is unavailable. Demo never supplies
PCM or audio history. Starting and stale capture remain quiet.

## Build and release checks

On Linux, run `go mod verify`, `go vet ./...`, `go test ./...`,
`go test -race ./...` and both cgo/nocgo builds. On macOS 14.2+ with the Xcode
Command Line Tools, `make verify` also runs the native cgo conversion, SPSC ring,
timeout/teardown tests and checks the Mach-O minimum deployment version.
The `verify` development workflow runs these checks and packages all six targets
on macOS. The Windows workflow retains native browser/installer checks.

Permissioned macOS hardware capture is a separate gate, requiring System Audio
Recording permission for the launching terminal/application. With controlled
audio playing, run:

```sh
AMTUI_TEST_COREAUDIO=1 AMTUI_TEST_COREAUDIO_REQUIRE_SIGNAL=1 \
  go test -race -v -count=1 -timeout=30s ./visualizer -run '^TestCoreAudioLive'
```

Repeat paused without `REQUIRE_SIGNAL`; fresh zero PCM is valid silence. An
opened tap with no received PCM now fails the smoke test. Record permission
denial, cancellation, close/reopen and playing/paused/resumed results on the
actual Mac before calling live capture verified. Hosted unit tests do not prove
permissioned hardware capture or audible subscription playback.

`make dist VERSION=...` builds Linux/Windows on Linux and adds native cgo macOS
archives on macOS. It includes README, upstream LICENSE, all pinned dependency
notices, these migration notes, and the example config, then writes SHA256SUMS.
On Linux it explicitly skips macOS; a Darwin nocgo cross-build is not a substitute
release. No release tag or published release is created by this candidate.

Regenerate notices using `GO=/path/to/go python3 scripts/license-notices.py`.
It preserves all supplied license/copyright files for the 29 pinned modules,
Gonum's incorporated algorithm notices, and the build toolchain/runtime license.
`go-localereader v0.0.1` provides its MIT declaration and author in README rather
than a standalone license file; that exact README is included in the notices.
Chromium, Widevine, system fonts and Apple Music content are not bundled.
